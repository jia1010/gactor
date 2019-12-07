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
package gactor

import (
	"github.com/mafei198/gactor/actor"
	"github.com/mafei198/gactor/etcd"
	"github.com/mafei198/gactor/etcd/agents"
	"github.com/mafei198/goslib/logger"
)

var (
	actorMgr = new(actor.Manager)
	rpcMgr   = new(actor.RpcMgr)
)

func Start() error {
	if err := etcd.Start(); err != nil {
		return err
	}
	if err := actorMgr.Start(); err != nil {
		return err
	}

	port, err := rpcMgr.Start()
	if err != nil {
		return err
	}

	nodeAgent, err := agents.NewNodeAgent(port)
	if err != nil {
		return err
	}
	if err := nodeAgent.Start(); err != nil {
		return err
	}

	actorAgent := agents.NewActorAgent()
	if err := actorAgent.Start(); err != nil {
		return err
	}
	return nil
}

func Stop() {
	if err := actorMgr.Stop(); err != nil {
		logger.ERR("stop actorMgr failed: ", err)
	}
	if err := rpcMgr.Stop(); err != nil {
		logger.ERR("stop rpcMgr failed: ", err)
	}
}
