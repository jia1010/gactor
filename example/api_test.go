package main

import (
	"github.com/golang/protobuf/proto"
	"github.com/mafei198/gactor/example/protos"
	"github.com/mafei198/goslib/logger"
	"github.com/mafei198/goslib/pbmsg"
	"strconv"
	"testing"
	"time"
)

func TestRegister(t *testing.T) {
	pbmsg.Register(func() proto.Message { return &protos.Player{} })
	start := time.Now().UnixNano()
	for i := 0; i < 100000; i++ {
		msg := &protos.Player{
			Name: "savin",
			Age:  18,
		}
		data, err := pbmsg.Encode(msg)
		if err != nil {
			panic(err)
		}
		_, err = pbmsg.Decode(data)
		if err != nil {
			panic(err)
		}
	}
	logger.INFO("msg process", UsedMS(start, time.Now().UnixNano()))
}

func UsedMS(startNanoSecond, stopNanosecond int64) string {
	return "used " + strconv.Itoa(int((stopNanosecond-startNanoSecond)/1000000)) + "ms"
}
