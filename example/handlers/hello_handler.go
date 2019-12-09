package handlers

import (
	"github.com/golang/protobuf/proto"
	"github.com/mafei198/gactor/api"
	"github.com/mafei198/gactor/example/actors"
	"github.com/mafei198/gactor/example/gen/gd"
	"github.com/mafei198/gactor/example/protos"
	"github.com/mafei198/goslib/logger"
)

type HelloHandler struct{}

var Hello = new(HelloHandler)

func init() {
	// handlers
	actors.Player.Register(new(protos.Player), Hello.Login)
	actors.Player.Register(new(protos.Equip), Hello.Equip)
}

func (*HelloHandler) Ctx(ctx interface{}) *actors.PlayerBehavior {
	return ctx.(*actors.PlayerBehavior)
}

func (h *HelloHandler) Login(req *api.Request) proto.Message {
	params := req.Params.(*protos.Player)
	conf := gd.HeroIns.GetList()[0]
	logger.INFO("heros: ", conf)
	logger.INFO(params.Name)
	return params
}

func (h *HelloHandler) Equip(req *api.Request) proto.Message {
	params := req.Params.(*protos.Equip)
	logger.INFO(params.Name)
	return params
}
