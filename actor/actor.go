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
	"gactor/actor/gen_server"
	"gactor/api"
	"gactor/logger"
	"gactor/utils"
	"time"
)

type WrapHandler func(ctx interface{}) interface{}
type AsyncWrapHandler func(ctx interface{})

const ExpireDuration = 600 // seconds

type Behavior interface {
	OnStart(server *Server) error
	//OnCall(msg interface{}) (interface{}, error)
	//OnCast(msg interface{})
	OnStop(reason string) error
}

func Call(actorId string, msg interface{}, options ...*gen_server.Option) (interface{}, error) {
	server, err := GetActor(actorId)
	if err != nil {
		return nil, err
	}
	return server.Call(msg, options...)
}

func Cast(actorId string, msg interface{}) error {
	server, err := GetActor(actorId)
	if err != nil {
		return err
	}
	return server.Cast(msg)
}

func Wrap(id string, handler WrapHandler, options ...*gen_server.Option) (interface{}, error) {
	return Call(id, &wrapParams{Handler: handler}, options...)
}

func AsyncWrap(id string, handler AsyncWrapHandler) error {
	return Cast(id, &asyncWrapParams{Handler: handler})
}

func Request(accountId string, request *api.Request) error {
	return Cast(accountId, &requestParams{request: request})
}

type Server struct {
	Meta      *Meta
	PlayerId  string
	SceneId   string
	Factory   *Factory
	Actor     Behavior
	ActiveAt  int64
	Processed int64

	tickers []*time.Ticker
}

type requestParams struct{ request *api.Request }
type wrapParams struct{ Handler WrapHandler }
type asyncWrapParams struct{ Handler AsyncWrapHandler }

func (ins *Server) Init(args []interface{}) (err error) {
	ins.Meta = args[0].(*Meta)
	ins.Factory = args[1].(*Factory)
	ins.Actor = ins.Factory.Constructor()
	ins.PlayerId = ins.Meta.Uuid
	ins.ActiveAt = time.Now().Unix()
	ins.StartTicker(time.Minute, &activeCheckParams{})
	return ins.Actor.OnStart(ins)
}

type activeCheckParams struct{}

func (ins *Server) HandleCall(req *gen_server.Request) (interface{}, error) {
	switch req.Msg.(type) {
	case *activeCheckParams:
		if time.Now().Unix()-ins.ActiveAt >= ExpireDuration {
			if !ins.Meta.Dispatch.IsDaemon {
				MarkActorSleep(ins.PlayerId)
			}
		}
		return nil, nil
	default:
		ins.ActiveAt = time.Now().Unix()
		handler, ok := ins.Factory.Route(req.Msg)
		if !ok || handler == nil {
			return nil, api.ErrRouteNotFound
		}
		request := api.NewLocalRequest(api.ReqCall, req.Msg)
		return handler(request), nil
	}
}

func (ins *Server) HandleCast(req *gen_server.Request) {
	ins.ActiveAt = time.Now().Unix()
	switch params := req.Msg.(type) {
	case *requestParams:
		_ = ins.handleRequest(params)
	case *wrapParams:
		params.Handler(ins.Actor)
	default:
		if handler, ok := ins.Factory.Route(req.Msg); ok && handler != nil {
			request := api.NewLocalRequest(api.ReqCast, req.Msg)
			handler(request)
		} else {
			logger.ERR("Msg: ", req.Msg, api.ErrRouteNotFound)
		}
	}
}

func (ins *Server) Terminate(reason string) error {
	err := ins.Actor.OnStop(reason)
	if err == nil {
		for _, ticker := range ins.tickers {
			ticker.Stop()
		}
		if !ins.Meta.Dispatch.IsDaemon {
			ExpireMeta(ins.Meta.Uuid)
		}
	}
	return err
}

func (ins *Server) handleRequest(params *requestParams) error {
	req := params.request
	ins.Processed++
	handler, ok := ins.Factory.Route(req.Params)
	if !ok || handler == nil {
		logger.ERR("Route not found: ", utils.GetType(req.Params))
		return api.ErrRouteNotFound
	}
	req.Ctx = ins.Actor
	if rsp := handler(req); rsp != nil {
		return req.Response(rsp)
	}
	return nil
}

func (ins *Server) GetActorId() string {
	return ins.Meta.Uuid
}

func (ins *Server) GetCategory() string {
	return ins.Meta.Category
}

func (ins *Server) StartTicker(duration time.Duration, msg interface{}) {
	ticker := time.NewTicker(duration)
	ins.tickers = append(ins.tickers, ticker)
	category := ins.Meta.Category
	actorId := ins.Meta.Uuid
	go func() {
		for range ticker.C {
			if _, err := gen_server.Call(actorId, msg); err != nil {
				logger.ERR("ticker failed: ", category, actorId, msg, err)
				if err == gen_server.ErrNotExist {
					break
				}
			}
		}
	}()
}
