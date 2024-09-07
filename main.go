package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/SnakebiteEF2000/go-client-improved/watcher"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type StatusCode = int

const (
	StatusOK StatusCode = iota
	StatusConfigError
	StatusClientError
)

// testing only
const Kubeconfig = "/home/thma/.kube/config"

var WG sync.WaitGroup

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

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(clusterClient, 5*time.Second, corev1.NamespaceAll, nil)
	informer := factory.ForResource(Resource).Informer()

	Waim := watcher.Watcher{}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ipchan := Waim.Watch(ctx, obj)
			t := <-ipchan
			fmt.Println("Name", t.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			Waim.Watch(ctx, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			Waim.Watch(ctx, obj)
		},
	})

	informer.Run(ctx.Done())

	return StatusOK
}
