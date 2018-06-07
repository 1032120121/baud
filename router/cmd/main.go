package main

import (
	"flag"
	"github.com/tiglabs/baudengine/util/log"
	"github.com/tiglabs/baudengine/router"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"sync"
	"fmt"
)

const (
	Version     = "0.1"
	LogicalCPUs = 32
)

var (
	configFile = flag.String("c", "", "config file path")

	mainWg sync.WaitGroup
)

func interceptSignal(s *router.Router) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	mainWg.Add(1)
	go func() {
		defer mainWg.Done()

		<-sigs
		s.Shutdown()
	}()
}

func main() {
	fmt.Println("Hello, BaudEngine! I router!!!")
	flag.Parse()

	cfg := router.LoadConfig(*configFile)

	log.InitFileLog(cfg.LogCfg.LogPath, cfg.ModuleCfg.Role, cfg.LogCfg.Level)
	defer log.Flush()
	log.Debug("log has been initialized")

	//for multi-cpu scheduling
	runtime.GOMAXPROCS(runtime.NumCPU())

	//init profile server
	go func() {
		fmt.Println(http.ListenAndServe(fmt.Sprintf(":%d", cfg.ModuleCfg.Pprof), nil))
	}()

	server := router.NewServer()

	//install the signal handler
	interceptSignal(server)

	//start the server
	err := server.Start(cfg)
	if err != nil {
		log.Fatal("Fatal: failed to start the router daemon - ", err)
	}

	log.Info("main waiting")
	mainWg.Wait()
	log.Info("main exit")
}
