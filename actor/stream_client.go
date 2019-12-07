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
	"github.com/mafei198/gactor/cluster"
	proto "github.com/mafei198/gactor/rpc_proto"
	"github.com/mafei198/goslib/gen_server"
	"github.com/mafei198/goslib/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
	"strings"
	"sync"
)

var gameStreamsMap = &sync.Map{}

const AGENT_SERVER = "__AGENT_SERVER__"

type Stream struct {
	GameAppId    string
	StreamClient proto.GameRpcServer_RpcStreamClient
	RpcClient    proto.GameRpcServerClient
}

type RpcRspHandler func(reqId int32, fromActorId string, data []byte)

var rpcRspHandler RpcRspHandler

func StartRpcClient(handler RpcRspHandler) error {
	rpcRspHandler = handler
	_, err := gen_server.Start(AGENT_SERVER, new(StreamManager))
	return err
}

// 创建连接
func GetStream(targetPlayerId string) (*Stream, error) {
	meta, err := GetMeta(targetPlayerId)
	if err != nil {
		return nil, err
	}
	node, ok := meta.GetNode()
	if !ok {
		return nil, errNodeNotFound
	}
	return GetStreamClient(node)
}

func GetStreamClient(game *cluster.Node) (*Stream, error) {
	if stream, ok := gameStreamsMap.Load(game.Uuid); ok {
		return stream.(*Stream), nil
	}
	stream, err := gen_server.Call(AGENT_SERVER, &ConnectGameAppParams{game})
	if err != nil {
		return nil, err
	}
	return stream.(*Stream), nil
}

/*
   GenServer Callbacks
*/
type StreamManager struct {
}

func (m *StreamManager) Init([]interface{}) (err error) {
	return nil
}

func (m *StreamManager) HandleCast(req *gen_server.Request) {
}

type ConnectGameAppParams struct {
	game *cluster.Node
}

func (m *StreamManager) HandleCall(req *gen_server.Request) (interface{}, error) {
	switch params := req.Msg.(type) {
	case *ConnectGameAppParams: // 连接其他GameServer的rpc服务器
		game := params.game
		if conn, ok := gameStreamsMap.Load(game.Uuid); ok {
			return conn, nil
		}
		addr := strings.Join([]string{game.RpcHost, game.RpcPort}, ":")
		return m.doConnectGameApp(game.Uuid, addr)
	}
	return nil, nil
}

func (m *StreamManager) Terminate(reason string) (err error) {
	return nil
}

// 连接其他GameServer的rpc服务器
func (m *StreamManager) doConnectGameApp(gameAppId string, addr string) (*Stream, error) {
	logger.INFO("ConnectGameApp: ", addr)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		logger.ERR("did not connect: ", err)
		return nil, err
	}
	// 得到rpc Client实例
	client := proto.NewGameRpcServerClient(conn)
	header := metadata.New(map[string]string{
		"nodeId":       gameAppId,
		"clientnodeid": cluster.GetCurrentNodeId(),
	})
	ctx := metadata.NewOutgoingContext(context.Background(), header)
	streamClient, err := client.RpcStream(ctx)
	stream := &Stream{
		GameAppId:    gameAppId,
		StreamClient: streamClient,
		RpcClient:    client,
	}
	if err != nil {
		logger.ERR("startPlayerProxyStream failed: ", client, " err:", err)
		return nil, err
	}
	startReceiver(stream)
	gameStreamsMap.Store(gameAppId, stream)
	return stream, nil
}

func startReceiver(stream *Stream) {
	go func() {
		for {
			in, err := stream.StreamClient.Recv()
			if err == io.EOF {
				logger.ERR("AgentStream read done: ", err)
				break
			}
			if err != nil {
				logger.ERR("AgentStream failed to receive : ", err)
				break
			}
			rpcRspHandler(in.ReqId, in.FromActorId, in.Data)
		}
		gameStreamsMap.Delete(stream.GameAppId)
		err := stream.StreamClient.CloseSend()
		if err != nil {
			logger.ERR("proxy close stream failed: ", err)
		}
	}()
}
