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

type IppoolRes struct {
	Name  string
	Start string
	End   string
}

func (wa *Watcher) IsHealthy() bool {
	return wa.healthy.Load()
}

func (wa *Watcher) Watch(ctx context.Context, obj interface{}) <-chan IppoolRes {
	ipoolChan := make(chan IppoolRes)

	go processIpool(ctx, toUnstructured(obj), ipoolChan)

	wa.healthy.Store(true)
	return ipoolChan
}

func processIpool(ctx context.Context, obj *unstructured.Unstructured, c chan IppoolRes) error {
	select {
	default:
		name, startip, endip, err := getIPPoolInformation(obj)
		if err != nil {
			log.Println(err)
			return err
		}
		t := IppoolRes{
			Name:  name,
			Start: startip,
			End:   endip,
		}
		c <- t
		return nil
	case <-ctx.Done():
		return nil
	}
}

func toUnstructured(obj interface{}) *unstructured.Unstructured {
	return obj.(*unstructured.Unstructured)
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
