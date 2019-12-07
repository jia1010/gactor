package protos

import (
	"github.com/golang/protobuf/proto"
	"github.com/mafei198/goslib/pbmsg"
)

func init() {
	pbmsg.Register(func() proto.Message { return new(Player) })
	pbmsg.Register(func() proto.Message { return new(Equip) })
}