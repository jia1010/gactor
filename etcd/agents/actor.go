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
package agents

import (
	"context"
	"gactor/actor"
	"gactor/cluster"
	"gactor/etcd"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
	"time"
)

type ActorAgent struct{}

func NewActorAgent() *ActorAgent {
	return new(ActorAgent)
}

func (a *ActorAgent) Start() error {
	var err error
	go func() {
		err = a.Watch()
	}()
	return err
}

func (a *ActorAgent) Watch() error {
	metas := map[string]*actor.Meta{}
	actor.CacheMetas = metas
	ch := etcd.Client.Watch(context.Background(), cluster.NodePrefix(), clientv3.WithPrefix())
	for result := range ch {
		for _, event := range result.Events {
			switch event.Type {
			case mvccpb.PUT:
				actor.StoreMeta(metas, event.Kv)
			case mvccpb.DELETE:
				actor.DelMetaCache(string(event.Kv.Key), event.Kv.ModRevision)
			}
		}
	}
	a.retryWatch()
	return nil
}

func (a *ActorAgent) retryWatch() {
	var err error
	for {
		go func() {
			err = a.Watch()
		}()
		if err != nil {
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}
}
