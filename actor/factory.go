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
	"gactor/actor/gen_server"
	"gactor/cluster"
	proto "gactor/rpc_proto"
	"time"
)

type Factory struct {
	Category    string
	Constructor func() IActor
}

var Factories = map[string]*Factory{}

func NewFactory(category string, factory func() IActor) *Factory {
	actorAgent := &Factory{Category: category, Constructor: factory}
	Factories[category] = actorAgent
	return actorAgent
}

func GetFactory(category string) *Factory {
	return Factories[category]
}

func (factory *Factory) Create(id, serverId string, dispatch *Dispatch) (*Meta, error) {
	return AddMeta(factory.Category, id, serverId, dispatch)
}

func (factory *Factory) StartService(id, serverId string, dispatch *Dispatch, options ...*gen_server.Option) error {
	meta, err := FindOrCreate(factory.Category, id, serverId, dispatch)
	if err != nil {
		return err
	}
	if meta.NodeId == cluster.GetCurrentNodeId() {
		_, err = StartActor(id, options...)
		return err
	} else {
		client, err := GetStream(meta.Uuid)
		if err != nil {
			return err
		}
		var timeout time.Duration
		if len(options) > 0 && options[0].Timeout > 0 {
			timeout = options[0].Timeout
		} else {
			timeout = gen_server.GetTimeout()
		}
		timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_, err = client.RpcClient.StartActor(timeoutCtx, &proto.StartActorReq{
			ActorId: meta.Uuid,
			Timeout: int64(timeout),
		})
		return err
	}
}
