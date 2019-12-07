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
	"github.com/mafei198/gactor/api"
	"github.com/mafei198/gactor/cluster"
	rpcproto "github.com/mafei198/gactor/rpc_proto"
	"github.com/mafei198/goslib/gen_server"
	"github.com/mafei198/goslib/pbmsg"
	"math"
	"sync/atomic"
	"time"
)

type RpcHandler func(ctx interface{}, rsp interface{}, err error) error

type RpcClient struct {
	Client rpcproto.GameRpcServerClient
}

type RpcRequest struct {
	*rpcproto.StreamAgentMsg
	Params    interface{}
	CreatedAt int64
	Handler   RpcHandler
	Req       *gen_server.Request
}

var rpcRequestId int64

func (r *RpcRequest) IsLocal() (bool, error) {
	meta, err := GetMeta(r.ToActorId)
	if err != nil {
		return false, err
	}
	return meta.NodeId == cluster.GetCurrentNodeId(), nil
}

// 异步发送RPC消息
func RpcCast(toActorId string, params interface{}) error {
	request, err := newRpcRequest(api.ReqCast, "", toActorId, params)
	if err != nil {
		return err
	}
	return sendRequest(request)
}

// 同步Call
func RpcCall(toActorId string, params interface{}) (interface{}, error) {
	request, err := newRpcRequest(api.ReqCall, "", toActorId, params)
	if err != nil {
		return nil, err
	}
	if err := sendRequest(request); err != nil {
		return nil, err
	}
	return WaitForRpcRequest(request)
}

// 异步发送RPC消息，并异步回调结果
func RpcAsyncCall(fromActorId, toActorId string, params interface{}, callback RpcHandler) error {
	request, err := newRpcRequest(api.ReqCall, fromActorId, toActorId, params)
	if err != nil {
		return err
	}
	request.Handler = callback
	err = sendRequest(request)
	if err == nil {
		return AddRpcRequest(request)
	}
	return err
}

func sendRequest(request *RpcRequest) error {
	isLocal, err := request.IsLocal()
	if err != nil {
		return err
	}
	if isLocal {
		agent := GetStreamAgent(cluster.GetCurrentNodeId())
		return agent.LocalRequest(request)
	} else {
		stream, err := GetStream(request.ToActorId)
		if err != nil {
			return err
		}
		data, err := pbmsg.Encode(request.Params)
		if err != nil {
			return err
		}
		request.StreamAgentMsg.Data = data
		return stream.StreamClient.Send(request.StreamAgentMsg)
	}
}

func newRpcRequest(reqType int32, fromActorId, toActorId string, msg interface{}) (*RpcRequest, error) {
	reqId := genRpcReqId()
	agentMsg := &rpcproto.StreamAgentMsg{
		ReqId:       reqId,
		ReqType:     reqType,
		FromActorId: fromActorId,
		ToActorId:   toActorId,
	}
	return &RpcRequest{
		StreamAgentMsg: agentMsg,
		Params:         msg,
		CreatedAt:      time.Now().Unix(),
	}, nil
}

func genRpcReqId() int32 {
	return int32(atomic.AddInt64(&rpcRequestId, 1) % math.MaxInt32)
}
