package handlers

import (
	"github.com/golang/protobuf/proto"
	"github.com/mafei198/gactor/api"
	"github.com/mafei198/gactor/example/actors"
	"github.com/mafei198/gactor/example/protos"
	"github.com/mafei198/gactor/logger"
)

type HelloHandler struct{}

var Hello = new(HelloHandler)

func init() {
	// messages
	api.Register(func() proto.Message { return new(protos.Player) })
	api.Register(func() proto.Message { return new(protos.Equip) })

	// handlers
	actors.Player.Register(new(protos.Player), Hello.Login)
	actors.Player.Register(new(protos.Equip), Hello.Equip)
}

func (*HelloHandler) Ctx(ctx interface{}) *actors.PlayerBehavior {
	return ctx.(*actors.PlayerBehavior)
}

func (h *HelloHandler) Login(req *api.Request) proto.Message {
	params := req.Params.(*protos.Player)
	logger.INFO(params.Name)
	return params
}

func (h *HelloHandler) Equip(req *api.Request) proto.Message {
	params := req.Params.(*protos.Equip)
	logger.INFO(params.Name)
	return params
}
