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
	"context"
	"errors"
	"fmt"
	"github.com/mafei198/gactor/actor/gen_server"
	"github.com/mafei198/gactor/cluster"
	"github.com/mafei198/gactor/logger"
	proto "github.com/mafei198/gactor/rpc_proto"
	"github.com/rs/xid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"sync"
	"time"
)

type RpcAgentServer struct{}

var streamListener net.Listener
var agents = &sync.Map{}

func StartRpcServer() (int, error) {
	var err error
	streamListener, err = net.Listen("tcp4", ":0")
	if err != nil {
		logger.ERR("failed to listen: ", err)
		return 0, err
	}
	port := streamListener.Addr().(*net.TCPAddr).Port
	logger.INFO("StreamServer lis: tcp4", " port: ", port)
	grpcServer := grpc.NewServer()
	proto.RegisterGameRpcServerServer(grpcServer, &RpcAgentServer{})
	go func() {
		err = grpcServer.Serve(streamListener)
	}()
	return port, err
}

func AddLocalAgent() {
	// 添加本地Agent
	AddStreamAgent(cluster.GetCurrentNodeId(), &RpcStream{
		uuid:         xid.New().String(),
		clientNodeId: cluster.GetCurrentNodeId(),
	})
}

func AddStreamAgent(clientNodeId string, agent *RpcStream) {
	id := fmt.Sprintln(clientNodeId, "->", cluster.GetCurrentNodeId())
	agents.Store(id, agent)
}

func GetStreamAgent(clientNodeId string) *RpcStream {
	id := fmt.Sprintln(clientNodeId, "->", cluster.GetCurrentNodeId())
	if agent, ok := agents.Load(id); ok {
		return agent.(*RpcStream)
	}
	return nil
}

/*
	handle player requests which proxied from other game servers
*/
func (s *RpcAgentServer) RpcStream(stream proto.GameRpcServer_RpcStreamServer) error {
	headers, _ := metadata.FromIncomingContext(stream.Context())
	nodeId := headers["nodeid"][0]
	clientNodeId := headers["clientnodeid"][0]
	if cluster.GetCurrentNodeId() != nodeId {
		errMsg := fmt.Sprintln("connect wrong node, expect: ", nodeId, " current: ", cluster.GetCurrentNodeId())
		return errors.New(errMsg)
	}
	return NewRpcStream(clientNodeId, stream).receiveLoop()
}

/*
	handle player requests from agent
*/
func (s *RpcAgentServer) AgentStream(stream proto.GameRpcServer_AgentStreamServer) error {
	headers, _ := metadata.FromIncomingContext(stream.Context())
	nodeId := headers["nodeid"][0]
	accountId := headers["accountid"][0]
	if cluster.GetCurrentNodeId() != nodeId {
		errMsg := fmt.Sprintln("connect wrong node, expect: ", nodeId, " current: ", cluster.GetCurrentNodeId())
		return errors.New(errMsg)
	}
	if accountId == "" {
		return errors.New("actorId is blank")
	}
	return NewAgentStream(accountId, stream).receiveLoop()
}

func (s *RpcAgentServer) StartActor(ctx context.Context, in *proto.StartActorReq) (*proto.StartActorRsp, error) {
	var timeout time.Duration
	if in.Timeout > 0 {
		timeout = time.Duration(in.Timeout)
	} else {
		timeout = gen_server.GetTimeout()
	}
	_, err := StartActor(in.ActorId, &gen_server.Option{Timeout: timeout})
	return &proto.StartActorRsp{Success: err == nil}, err
}
