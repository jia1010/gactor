package handlers

import (
	"gactor/api"
	"gactor/example/actors"
	"gactor/example/protos"
	"gactor/logger"
	"github.com/golang/protobuf/proto"
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
