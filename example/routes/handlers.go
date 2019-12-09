package routes

import (
	"github.com/mafei198/gactor/example/actors"
	"github.com/mafei198/gactor/example/gen/pt"
	"github.com/mafei198/gactor/example/handlers"
)

/*
 */
func init() {
	// handlers
	actors.Player.Register(new(pt.Player), handlers.Hello.Login)
	actors.Player.Register(new(pt.Equip), handlers.Hello.Equip)
}
