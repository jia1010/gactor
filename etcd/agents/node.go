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
	"encoding/json"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/mafei198/gactor/actor"
	"github.com/mafei198/gactor/cluster"
	"github.com/mafei198/gactor/etcd"
	"github.com/mafei198/goslib/logger"
	"github.com/mafei198/goslib/misc"
	"go.etcd.io/etcd/clientv3"
	"strconv"
	"sync"
	"time"
)

type NodeAgent struct {
	Node *cluster.Node
}

func NewNodeAgent(port int) (*NodeAgent, error) {
	localIP, err := misc.GetLocalIp()
	if err != nil {
		return nil, err
	}
	node := cluster.NewNode(cluster.RoleDefault, localIP, strconv.Itoa(port))
	return &NodeAgent{
		Node: node,
	}, nil
}

func (n *NodeAgent) Start() error {
	cluster.SetCurrentNodeId(n.Node.Uuid)
	actor.AddLocalAgent()
	var err error
	go func() {
		err = n.Watch()
	}()
	if err != nil {
		return err
	}
	return n.KeepAlive(n.Node)
}

func (n *NodeAgent) Watch() error {
	nodes := &sync.Map{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	rsp, err := etcd.Client.Get(ctx, cluster.NodePrefix(), clientv3.WithPrefix())
	cancel()
	if err != nil {
		return err
	}
	for _, kv := range rsp.Kvs {
		n.storeNode(nodes, kv)
	}
	cluster.CacheNodes = nodes
	ch := etcd.Client.Watch(context.Background(), cluster.NodePrefix(), clientv3.WithPrefix())
	for result := range ch {
		for _, event := range result.Events {
			switch event.Type {
			case mvccpb.PUT:
				n.storeNode(nodes, event.Kv)
			case mvccpb.DELETE:
				nodes.Delete(string(event.Kv.Key))
			}
		}
	}
	n.retryWatchNodes()
	return nil
}

func (n *NodeAgent) retryWatchNodes() {
	var err error
	for {
		go func() {
			err = n.Watch()
		}()
		if err != nil {
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}
}

func (n *NodeAgent) storeNode(nodes *sync.Map, kv *mvccpb.KeyValue) {
	node := &cluster.Node{}
	err := json.Unmarshal(kv.Value, node)
	if err == nil {
		nodes.Store(string(kv.Key), node)
	} else {
		logger.ERR("unmarshal node failed: ", err, string(kv.Value))
	}
}

func (n *NodeAgent) KeepAlive(node *cluster.Node) error {
	lease, err := etcd.Client.Grant(context.TODO(), 5)
	if err != nil {
		return err
	}
	go func() {
		ch, err := etcd.Client.KeepAlive(context.TODO(), lease.ID)
		if err != nil {
			logger.ERR("keepalive failed: ", err)
		}
		for ka := range ch {
			logger.INFO("node ttl: ", ka)
			updateNode(etcd.Client, node, lease)
		}
		logger.ERR("keepalive channel closed!")
		n.RetryKeepAlive(node)
	}()
	return err
}

func (n *NodeAgent) RetryKeepAlive(node *cluster.Node) {
	for {
		err := n.KeepAlive(node)
		if err == nil {
			break
		} else {
			logger.ERR("RetryKeepAlive failed: ", err)
		}
	}
}

func updateNode(cli *clientv3.Client, node *cluster.Node, lease *clientv3.LeaseGrantResponse) {
	ccu, err := actor.GetActorAmount()
	if err != nil {
		logger.ERR("get actor amount failed: ", err)
	}
	node.Ccu = ccu
	data, err := json.Marshal(node)
	if err != nil {
		logger.ERR("marshal node failed: ", err, node)
	}
	_, err = cli.Put(context.TODO(), node.Uuid, string(data), clientv3.WithLease(lease.ID))
	misc.PrintMemUsage()
}
