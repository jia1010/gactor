package main

import (
	"gactor"
	"gactor/actor"
	"gactor/api"
	"gactor/example/actors"
	"gactor/example/protos"
	"gactor/logger"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := gactor.Start(); err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)
	sendExampleMsg()

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	<-stopChan // wait for SIGINT or SIGTERM
	logger.INFO("Shutting gactor ...")
	gactor.Stop()
}

func sendExampleMsg() {
	// register msg factory
	api.RegisterFactory(func() proto.Message {
		return &protos.Player{}
	})
	// register handler
	api.RegisterHandler(&protos.Player{}, func(req *api.Request) proto.Message {
		logger.INFO("handle request: ", req)
		return req.Params.(*protos.Player)
	})
	// create actor
	actorId := createActor()
	// send msg to actor
	err := gactor.Cast(actorId, &protos.Player{
		Name: "savin",
		Age:  21,
	})
	if err != nil {
		panic(err)
	}
}

func createActor() string {
	metaId := actor.GenMetaId()
	meta, err := actors.PlayerFactory.Create(metaId, "fake_server_id", &actor.Dispatch{
		Type:     actor.DispatchTypeDefault,
		IsDaemon: false,
	})
	if err != nil {
		panic(err)
	}
	logger.INFO(meta)
	return metaId
}

func sendRequest(actorId string) error {
	return gactor.Cast(actorId, &protos.Player{
		Name: "savin",
		Age:  21,
	})
}

func benchmark() {
	clients := 1000
	requests := 10000
	for i := 0; i < clients; i++ {
		// create actor
		metaId := createActor()
		randSleep()
		go func() {
			for j := 0; j < requests; j++ {
				randSleep()
				// send request
				if err := sendRequest(metaId); err != nil {
					panic(err)
				}
			}
		}()
	}
}

func randSleep() {
	var duration = 1 * time.Second
	time.Sleep(time.Duration(rand.Int63n(int64(duration))))
}
