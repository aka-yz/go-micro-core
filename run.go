package go_micro_core

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	// for stop()
	exitChan chan bool
	// CustomErrCh setup services
	CustomErrCh chan error

	HttpErrCh chan error
)

const (
	stopOrderKey = "injects_objects_stop_order"
)

var defaultStopOrder = []string{
	"*grpc.RPCServer",
	"*http.HTTPServer",
}

func Run(objs ...interface{}) {
	flag.Parse()
	// step-1: initial env, config, log configuration
	initEnv()
	InitConfig()
	initLog()

	// step-2: register providers(objs)
	RegisterProvider(objs...)
	p := NewProvider(injects.Vals...)
	if err := injects.Provide(p.Provide()...); err != nil {
		panic(err)
	}

	// step-3: populate objects
	if err := injects.Populate(); err != nil {
		panic(err)
	}

	// step-4: execute life-cycle flow
	fmt.Println("injects.Objects init...")
	for _, v := range injects.Objects() {
		if o, ok := v.Value.(initialization); ok {
			fmt.Println(v.String(), " init...")
			o.Init()
		}
	}

	fmt.Println("injects.Objects start...")
	for _, v := range injects.Objects() {
		if o, ok := v.Value.(starter); ok {
			fmt.Println(v.String(), " start...")
			o.Start()
		}
	}

	//server := handler.NewServer(customErrCh)
	//
	// start http server

	//go func() {
	//	httpErrCh <- server.ListenAndServe()
	//}()

	// Monitor system signal like INT or KILL
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT)

	select {
	case s := <-sigint:
		Logger.Warn().Msgf("received signal %s; shutting down", s)
		_, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		//if err := server.GracefulShutDown(ctx); err != nil {
		//	Logger.Fatal().Msgf("Server Shutdown on SIGINT: %s", err)
		//}
	case err := <-HttpErrCh:
		if err != nil && err != http.ErrServerClosed {
			Logger.Fatal().Msgf("HTTP server error: %s\n", err)
		}
	case err := <-CustomErrCh:
		if err != nil {
			Logger.Fatal().Msgf("custom error: %s", err)
		}
	}

	exitChan = make(chan bool, 1)
	stopObjectInOrder()
}

func Stop() {
	exitChan <- true
}

func stopObjectInOrder() {
	var stopOrder []string
	if err := conf.Get(stopOrderKey).Populate(&stopOrder); err != nil {
		panic(err)
	}
	if len(stopOrder) == 0 {
		stopOrder = defaultStopOrder
	}

	stoperByName := make(map[string]stoper)
	for _, v := range injects.Objects() {
		if o, ok := v.Value.(stoper); ok {
			fmt.Println(v.String(), "stoper collected")
			stoperByName[v.String()] = o
		}
	}

	for _, name := range stopOrder {
		if obj, ok := stoperByName[name]; ok {
			fmt.Println(name, "stop by name...")
			obj.Stop()
			delete(stoperByName, name)
		} else {
			fmt.Println("no stoper found by name: ", name)
		}
	}
	for name, obj := range stoperByName {
		fmt.Println(name, "stop...")
		obj.Stop()
	}
}
