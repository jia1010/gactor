package main

import (
	"github.com/mafei198/gactor"
	"github.com/mafei198/gactor/actor"
	"github.com/mafei198/gactor/example/actors"
	"github.com/mafei198/gactor/example/protos"
	"github.com/mafei198/goslib/logger"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mafei198/gactor/example/handlers"
	_ "github.com/mafei198/gactor/example/routes"
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
	// create actor
	meta, err := actors.Player.Create()
	if err != nil {
		panic(err)
	}
	// send msg to actor
	err = gactor.RpcCast(meta.Uuid, &protos.Player{
		Name: "savin",
		Age:  21,
	})
	if err != nil {
		panic(err)
	}
}

func createActor() string {
	metaId := actor.GenMetaId()
	meta, err := actors.Player.Create(metaId)
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
