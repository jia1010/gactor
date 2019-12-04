/*
The MIT License (MIT)

Copyright (c) 2018 SavinMax. All rights reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package actor

import (
	"errors"
	"gactor/actor/gen_server"
	"gactor/logger"
	"time"
)

type RpcMgr struct {
	rpcRequests     map[int32]*RpcRequest
	syncRpcRequests map[int32]*gen_server.Request
	ticker          *time.Ticker
}

const serverName = "__rpc_mgr__"

var server *gen_server.GenServer

func (m *RpcMgr) Start() (port int, err error) {
	if port, err = StartRpcServer(); err != nil {
		return
	}
	if err = m.StartClient(); err != nil {
		return
	}
	return
}

func (m *RpcMgr) StartClient() (err error) {
	if server, err = gen_server.Start(serverName, m); err != nil {
		return err
	}
	return StartRpcClient(OnRpcRsp)
}

func (m *RpcMgr) Stop() error {
	return gen_server.Stop(serverName, "shutdown")
}

func OnRpcRsp(reqId int32, fromActorId string, data []byte) {
	err := server.Cast(&RpcRspParams{
		ReqId:     reqId,
		AccountId: fromActorId,
		Data:      data,
	})
	if err != nil {
		logger.ERR("OnRpcRsp failed: ", err)
	}
}

func AddRpcRequest(request *RpcRequest) error {
	return server.Cast(&AddRpcParams{rpcRequest: request})
}

func WaitForRpcRequest(request *RpcRequest) (interface{}, error) {
	rsp, err := server.ManualCall(&AddRpcParams{rpcRequest: request})
	return rsp, err
}

func (m *RpcMgr) Init([]interface{}) (err error) {
	m.rpcRequests = map[int32]*RpcRequest{}
	ticker := time.NewTicker(1 * time.Second)
	m.ticker = ticker
	msg := &CheckTimeoutParams{}
	go func() {
		var err error
		for range ticker.C {
			_, err = gen_server.Call(serverName, msg)
			if err != nil {
				logger.ERR("rpc mgr ticker failed: ", err)
			}
		}
	}()
	return nil
}

type CheckTimeoutParams struct{}

func (m *RpcMgr) HandleCall(req *gen_server.Request) (interface{}, error) {
	switch params := req.Msg.(type) {
	case *CheckTimeoutParams:
		now := time.Now().Unix()
		for _, req := range m.rpcRequests {
			if time.Duration(now-req.CreatedAt)*time.Second >= gen_server.GetTimeout() {
				m.rpcTimeout(req)
			}
		}
		return nil, nil
	case *AddRpcParams:
		params.rpcRequest.Req = req
		m.addRpcRequest(params)
	}
	return nil, nil
}

func (m *RpcMgr) HandleCast(req *gen_server.Request) {
	switch params := req.Msg.(type) {
	case *RpcRspParams:
		m.rpcRsp(params)
	case *AddRpcParams:
		m.addRpcRequest(params)
	}
}

func (m *RpcMgr) Terminate(reason string) (err error) {
	if m.ticker != nil {
		m.ticker.Stop()
	}
	logger.INFO("RpcMgr terminate:", reason)
	return nil
}

type RpcRspParams struct {
	ReqId     int32
	AccountId string
	Data      []byte
}

func (m *RpcMgr) rpcRsp(params *RpcRspParams) {
	if req := m.getRpcRequest(params.ReqId); req != nil {
		m.delRpcRequest(params.ReqId)
		if req.Handler != nil {
			_ = asyncWrap(req.FromActorId, func(ctx interface{}) {
				if err := req.Handler(ctx, params, nil); err != nil {
					logger.ERR("handle rpc response failed: ", err)
				}
			})
		} else {
			req.Req.Response(params, nil)
		}
	}
}

var ErrTimeout = errors.New("rpc timeout")

func (m *RpcMgr) rpcTimeout(req *RpcRequest) {
	logger.ERR("Rpc timeout: ", req)
	m.delRpcRequest(req.ReqId)
	if req.Handler != nil {
		_ = asyncWrap(req.FromActorId, func(ctx interface{}) {
			if err := req.Handler(ctx, nil, ErrTimeout); err != nil {
				logger.ERR("handle rpc response failed: ", err)
			}
		})
	} else {
		req.Req.Response(nil, ErrTimeout)
	}
}

type AddRpcParams struct {
	rpcRequest *RpcRequest
}

func (m *RpcMgr) addRpcRequest(params *AddRpcParams) {
	m.rpcRequests[params.rpcRequest.ReqId] = params.rpcRequest
}

func (m *RpcMgr) getRpcRequest(rpcReqId int32) *RpcRequest {
	if req, ok := m.rpcRequests[rpcReqId]; ok {
		return req
	}
	return nil
}

func (m *RpcMgr) delRpcRequest(rpcReqId int32) {
	delete(m.rpcRequests, rpcReqId)
}
