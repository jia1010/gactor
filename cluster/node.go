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
package cluster

import (
	"github.com/rs/xid"
	"sort"
	"sync"
	"time"
)

const AllNodesKey = "{meta}.AllGameNodes"
const (
	RoleDefault = "GActorRoleDefault"
)

type Node struct {
	Uuid     string
	Role     string
	RpcHost  string
	RpcPort  string
	Ccu      int32
	ActiveAt int64
}

// Current server uuid
var currentNodeId string
var CacheNodes *sync.Map

func GetCurrentNodeId() string {
	return currentNodeId
}

func SetCurrentNodeId(id string) {
	currentNodeId = id
}

func FindNode(uuid string) (*Node, bool) {
	if node, ok := CacheNodes.Load(uuid); ok {
		return node.(*Node), ok
	} else {
		return nil, ok
	}
}

func NewNode(role, rpcHost, rpcPort string) *Node {
	uuid := NodePrefix() + xid.New().String()
	node := &Node{
		Uuid:     uuid,
		Role:     role,
		RpcHost:  rpcHost,
		RpcPort:  rpcPort,
		Ccu:      0,
		ActiveAt: time.Now().Unix(),
	}
	return node
}

func ChooseNode(role string) *Node {
	nodes := make([]*Node, 0)
	roleNodes := make([]*Node, 0)
	CacheNodes.Range(func(key, value interface{}) bool {
		node := value.(*Node)
		nodes = append(nodes, node)
		if node.Role == role {
			roleNodes = append(roleNodes, node)
		}
		return true
	})
	if node := recommend(roleNodes); node != nil {
		return node
	}
	return recommend(nodes)
}

func recommend(nodes []*Node) *Node {
	length := len(nodes)
	if length == 0 {
		return nil
	} else if length == 1 {
		return nodes[0]
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Ccu <= nodes[j].Ccu
	})
	return nodes[0]
}

func NodePrefix() string {
	return "{gactor}.Node:"
}
