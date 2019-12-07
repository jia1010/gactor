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
	"github.com/mafei198/goslib/logger"
	"github.com/mafei198/goslib/pbmsg"
	"github.com/rs/xid"
)

type RpcStream struct {
	uuid         string
	stream       rpcproto.GameRpcServer_RpcStreamServer
	clientNodeId string
}

func NewRpcStream(clientNodeId string, stream rpcproto.GameRpcServer_RpcStreamServer) *RpcStream {
	return &RpcStream{
		uuid:         xid.New().String(),
		stream:       stream,
		clientNodeId: clientNodeId,
	}
}

func (s *RpcStream) receiveLoop() error {
	AddStreamAgent(s.clientNodeId, s)
	// 开始接收消息
	var err error
	var in *rpcproto.StreamAgentMsg
	for {
		in, err = s.stream.Recv()
		if err != nil {
			logger.ERR("GameAgent err: ", err)
			break
		}
		if err = s.OnData(in); err != nil {
			break
		}
	}
	return err
}

func (s *RpcStream) OnData(in *rpcproto.StreamAgentMsg) error {
	rpcAgent := &RpcAgent{
		s:              s,
		StreamAgentMsg: in,
	}
	msg, err := pbmsg.Decode(in.Data)
	if err != nil {
		return err
	}
	request := api.NewRequest(rpcAgent, in.ReqType, in.ReqId, msg)
	return Request(in.ToActorId, request)
}

func (s *RpcStream) LocalRequest(in *RpcRequest) error {
	rpcAgent := &RpcAgent{
		s:              s,
		StreamAgentMsg: in.StreamAgentMsg,
	}
	request := api.NewRequest(rpcAgent, in.ReqType, in.ReqId, in.Params)
	return Request(in.ToActorId, request)
}

func (s *RpcStream) SendData(reqId int32, fromActorId string, data []byte) error {
	if s.clientNodeId == cluster.GetCurrentNodeId() {
		// 本地节点
		rpcRspHandler(reqId, fromActorId, data)
		return nil
	} else {
		// 远程节点
		return s.stream.Send(&rpcproto.StreamAgentRsp{
			ReqId:       reqId,
			FromActorId: fromActorId,
			Data:        data,
		})
	}
}
