package watcher

import (
	"context"
	"log"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Watcher struct {
	healthy atomic.Bool
}

func (wa *Watcher) IsHealthy() bool {
	return wa.healthy.Load()
}

func (wa *Watcher) Process(ctx context.Context, obj interface{}) error {
	//does stuff
	customResource, err := toUnstructured(obj)
	if err != nil {
		log.Println("Failed to case to unstructured")
		wa.healthy.Store(false)
	}
	name, startip, endip, err := getIPPoolInformation(customResource)
	if err != nil {
		wa.healthy.Store(false)
	}
	log.Printf("Name: %v", name)
	log.Printf("Range: %v - %v", startip, endip)

	wa.healthy.Store(true)
	return err
}

func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	return obj.(*unstructured.Unstructured), nil
}

func getIPPoolInformation(obj *unstructured.Unstructured) (string, string, string, error) {
	name := obj.GetName()
	spec, found, err := unstructured.NestedMap(obj.Object, "spec", "ipv4Config", "pool")
	if err != nil || !found {
		log.Printf("Error retrieving spec.pool: %v", err)
		return "", "", "", err
	}

	startIP, startFound, err := unstructured.NestedString(spec, "start")
	if err != nil || !startFound {
		startIP = "<unknown>"
	}

	endIP, endFound, err := unstructured.NestedString(spec, "end")
	if err != nil || !endFound {
		endIP = "<unknown>"
	}

	return name, startIP, endIP, nil
}
