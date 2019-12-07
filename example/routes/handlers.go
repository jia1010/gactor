package routes

import (
	"github.com/mafei198/gactor/example/actors"
	"github.com/mafei198/gactor/example/handlers"
	"github.com/mafei198/gactor/example/protos"
)

/*
 */
func init() {
	// handlers
	actors.Player.Register(new(protos.Player), handlers.Hello.Login)
	actors.Player.Register(new(protos.Equip), handlers.Hello.Equip)
}
