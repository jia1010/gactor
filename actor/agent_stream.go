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
	"github.com/mafei198/gactor/api"
	rpcproto "github.com/mafei198/gactor/rpc_proto"
	"github.com/mafei198/goslib/logger"
	"github.com/mafei198/goslib/pbmsg"
	"github.com/rs/xid"
)

type AgentStream struct {
	uuid        string
	stream      rpcproto.GameRpcServer_AgentStreamServer
	actorId     string
	closed      bool
	closeReason string
}

func NewAgentStream(actorId string, stream rpcproto.GameRpcServer_AgentStreamServer) *AgentStream {
	agent := &AgentStream{
		uuid:    xid.New().String(),
		stream:  stream,
		actorId: actorId,
	}
	return agent
}

func (a *AgentStream) receiveLoop() error {
	var err error
	var in *rpcproto.StreamAgentMsg
	for {
		if a.closed {
			return errors.New(a.closeReason)
		}
		in, err = a.stream.Recv()
		if err != nil {
			logger.ERR("GameAgent err: ", err)
			break
		}
		msg, err := pbmsg.Decode(in.Data)
		if err != nil {
			logger.ERR("AgentStream decode failed:", err)
			break
		}
		request := api.NewRequest(a, in.ReqType, in.ReqId, msg)
		if err := Request(a.actorId, request); err != nil {
			return err
		}
	}
	return err
}

func (a *AgentStream) GetUuid() string {
	return a.uuid
}

func (a *AgentStream) GetActorId() string {
	return a.actorId
}

func (a *AgentStream) SendData(reqId int32, data []byte) error {
	return a.stream.Send(&rpcproto.StreamAgentRsp{
		ReqId:       reqId,
		FromActorId: a.actorId,
		Data:        data,
	})
}

func (a *AgentStream) Close(reason string) error {
	a.closed = true
	a.closeReason = reason
	return nil
}
