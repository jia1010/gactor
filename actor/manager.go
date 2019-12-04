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
	"gactor/actor/gen_server"
	"gactor/cluster"
	"gactor/logger"
	"gactor/pool"
	"runtime"
	"time"
)

type Manager struct {
	status            int
	actors            map[string]*Factory
	categorisedActors map[string]map[string]*Factory
	sleeping          map[string]*SleepActor
	stopping          map[string]bool
	workerPool        *pool.Pool
}

const (
	MgrWorking = iota
	MgrStopping
)

type SleepActor struct {
	sleepAt int64
	server  *gen_server.GenServer
}

const actorMgrId = "ActorId:actor_mgr"

const (
	MaxSleep   = 30 // seconds
	GcInterval = 10 * time.Second
)

func (*Manager) Start() error {
	_, err := gen_server.Start(actorMgrId, new(Manager))
	return err
}

func (*Manager) Stop() error {
	if err := gen_server.Cast(actorMgrId, &shutdownActorsParams{}); err != nil {
		return err
	}
	for {
		result, err := GetActorAmount()
		logger.INFO("EnsureShutdown: ", result, err)
		if err == nil && result == 0 {
			break
		}
		if err != nil {
			logger.ERR("get remain actors failed: ", err)
		} else {
			logger.INFO("Stopping actors remain: ", result)
		}
		time.Sleep(1 * time.Second)
	}
	return gen_server.Stop(actorMgrId, "shutdown")
}

type startActorParams struct {
	ActorId string
}

func GetActor(actorId string, options ...*gen_server.Option) (*gen_server.GenServer, error) {
	server, ok := gen_server.GetGenServer(actorId)
	if !ok {
		return StartActor(actorId, options...)
	}
	return server, nil
}

func StartActor(actorId string, options ...*gen_server.Option) (*gen_server.GenServer, error) {
	value, err := gen_server.Call(actorMgrId, &startActorParams{
		ActorId: actorId,
	}, options...)
	if err != nil {
		return nil, err
	}
	return value.(*gen_server.GenServer), err
}

type shutdownActorsParams struct{}
type remainActorsParams struct{}

func GetActorAmount() (int32, error) {
	result, err := gen_server.Call(actorMgrId, &remainActorsParams{})
	if err == nil {
		return result.(int32), nil
	}
	return 0, err
}

type sleepParams struct{ actorId string }

func MarkActorSleep(actorId string) {
	_ = gen_server.Cast(actorMgrId, &sleepParams{actorId: actorId})
}

func (ins *Manager) Init([]interface{}) (err error) {
	ins.status = MgrWorking
	ins.workerPool, err = pool.New(runtime.NumCPU(), ins.workerHandler)
	if err != nil {
		return err
	}
	ins.actors = map[string]*Factory{}
	ins.sleeping = map[string]*SleepActor{}
	ins.stopping = map[string]bool{}
	ins.categorisedActors = map[string]map[string]*Factory{}
	ins.scheduleShutdownSleeps()
	return nil
}

type shutdownActorParams struct {
	actorId string
	server  *gen_server.GenServer
}

func (ins *Manager) workerHandler(msg interface{}) (interface{}, error) {
	switch params := msg.(type) {
	case *shutdownActorParams:
		err := params.server.Stop("shutdown")
		if err != nil {
			logger.ERR("shutdown: ", params.actorId, " failed: ", err)
			ins.workerPool.ProcessAsync(&shutdownActorParams{
				actorId: params.actorId,
				server:  params.server,
			})
			return nil, err
		}
		_ = gen_server.Cast(actorMgrId, &delActorParams{actorId: params.actorId})
	}
	return nil, nil
}

var locationErr = errors.New("actor not belongs to this server")
var shutDownErr = errors.New("actor manager is shutting down")

func (ins *Manager) HandleCall(req *gen_server.Request) (interface{}, error) {
	switch params := req.Msg.(type) {
	case *startActorParams:
		switch ins.status {
		case MgrWorking:
			return ins.handleStartActor(params.ActorId)
		case MgrStopping:
			return nil, shutDownErr
		}
	case *remainActorsParams:
		var amount int32
		for _, actors := range ins.categorisedActors {
			amount += int32(len(actors))
		}
		return amount, nil
	default:
		logger.ERR("Actor Manager unhandle call msg: ", req.Msg)
	}
	return nil, nil
}

func (ins *Manager) HandleCast(req *gen_server.Request) {
	switch params := req.Msg.(type) {
	case *sleepParams:
		ins.handleSleep(params.actorId)
		break
	case *shutdownSleepParams:
		ins.scheduleShutdownSleeps()
		break
	case *shutdownActorsParams:
		ins.status = MgrStopping
		ins.shutdownActors()
	case *delActorParams:
		if !gen_server.Exists(params.actorId) {
			ins.delActor(params.actorId)
		}
	default:
		logger.ERR("Actor Manager unhandle cast msg: ", req.Msg)
	}
}

func (ins *Manager) Terminate(reason string) (err error) {
	logger.ERR("Actor Manager Terminate:", reason)
	return nil
}

func (ins *Manager) handleStartActor(actorId string) (*gen_server.GenServer, error) {
	// return started actor
	if server, ok := gen_server.GetGenServer(actorId); ok {
		return server, nil
	}

	// wakeup sleeping actor
	if sleep, ok := ins.sleeping[actorId]; ok {
		gen_server.SetGenServer(actorId, sleep.server)
		return sleep.server, nil
	}

	// start actor
	meta, err := GetMeta(actorId)
	if err != nil {
		return nil, err
	}
	if meta.NodeId != cluster.GetCurrentNodeId() {
		logger.ERR("metaGameId: ", meta.NodeId, "nodeId: ", cluster.GetCurrentNodeId())
		return nil, locationErr
	}
	actorAgent := GetFactory(meta.Category)
	server, err := gen_server.Start(actorId, new(Server), meta, actorAgent.Constructor())
	ins.addActor(actorId, actorAgent)
	return server, err
}

func (ins *Manager) addActor(actorId string, actor *Factory) {
	actors, ok := ins.categorisedActors[actor.Category]
	if !ok {
		actors = map[string]*Factory{}
		ins.categorisedActors[actor.Category] = actors
	}
	actors[actorId] = actor
	ins.actors[actorId] = actor
}

type delActorParams struct{ actorId string }

func (ins *Manager) delActor(actorId string) {
	actor := ins.getActor(actorId)
	if actor == nil {
		return
	}
	if actors, ok := ins.categorisedActors[actor.Category]; ok {
		delete(actors, actorId)
	}
	delete(ins.actors, actorId)
	delete(ins.sleeping, actorId)
	delete(ins.stopping, actorId)
}

func (ins *Manager) getActor(actorId string) *Factory {
	if actorAgent, ok := ins.actors[actorId]; ok {
		return actorAgent
	}
	return nil
}

func (ins *Manager) actorExists(actorId string) bool {
	_, ok := ins.actors[actorId]
	return ok
}

type shutdownSleepParams struct{}

func (ins *Manager) scheduleShutdownSleeps() {
	now := time.Now().Unix()
	for actorId, sleep := range ins.sleeping {
		if now-sleep.sleepAt > MaxSleep {
			err := sleep.server.Stop("shutdown inactive")
			if err != nil {
				logger.ERR("shutdown inactive actor failed: ", actorId, err)
			} else {
				ins.delActor(actorId)
			}
		}
	}
	time.AfterFunc(GcInterval, func() {
		_ = gen_server.Cast(actorMgrId, &shutdownSleepParams{})
	})
}

func (ins *Manager) shutdownActors() {
	sleeped := 0
	for _, actors := range ins.categorisedActors {
		for actorId := range actors {
			if sleep, ok := ins.sleeping[actorId]; ok {
				ins.shutdownActor(actorId, sleep.server)
			} else {
				sleeped++
				ins.handleSleep(actorId)
			}
		}
	}
	if sleeped > 0 {
		_ = gen_server.Cast(actorMgrId, &shutdownActorsParams{})
	}
}

func (ins *Manager) shutdownActor(actorId string, server *gen_server.GenServer) {
	if _, ok := ins.stopping[actorId]; !ok {
		ins.stopping[actorId] = true
		ins.workerPool.ProcessAsync(&shutdownActorParams{
			actorId: actorId,
			server:  server,
		})
	}
}

func (ins *Manager) handleSleep(actorId string) {
	if ins.getActor(actorId) != nil {
		server, ok := gen_server.GetGenServer(actorId)
		if !ok {
			ins.delActor(actorId)
			return
		}
		gen_server.DelGenServer(actorId)
		ins.sleeping[actorId] = &SleepActor{
			sleepAt: time.Now().Unix(),
			server:  server,
		}
	}
}

func (ins *Manager) isActorSleep(actorId string) bool {
	_, ok := ins.sleeping[actorId]
	return ok
}
