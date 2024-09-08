package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/SnakebiteEF2000/go-client-improved/informer"
	"github.com/SnakebiteEF2000/go-client-improved/pooldataposter"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/SnakebiteEF2000/go-client-improved/monitor"
)

type StatusCode = int

const (
	StatusOK StatusCode = iota
	StatusConfigError
	StatusClientError
	StatusHTTPServerError
)

// testing only
const Kubeconfig = "/home/thma/.kube/config"

const resyncInterval = 5 * time.Minute

func main() {
	os.Exit(run())
}

func run() int {
	var clusterConfig *rest.Config
	var err error

	if Kubeconfig != "" {
		clusterConfig, err = clientcmd.BuildConfigFromFlags("", Kubeconfig)
	} else {
		clusterConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Println(err)
		return StatusConfigError
	}

	clusterClient, err := dynamic.NewForConfig(clusterConfig)
	if err != nil {
		log.Println(err)
		return StatusClientError
	}

	var watcher informer.Informer
	poster := pooldataposter.Poster{IPPoolInformer: &watcher}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		watcher.Run(ctx, clusterClient, resyncInterval)
	}()

	go func() {
		defer wg.Done()
		poster.Run(ctx)
	}()

	err = monitor.ListenAndServe(ctx, &poster, &watcher)
	if err != nil {
		slog.Error("http server failure",
			slog.String("err", err.Error()),
		)
	}

	wg.Wait()

	if err != nil {
		return StatusHTTPServerError
	}

	return StatusOK
}
