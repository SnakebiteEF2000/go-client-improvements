package informer

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"

	"github.com/SnakebiteEF2000/go-client-improved/pooldataposter"
)

var Resource = schema.GroupVersionResource{
	Group:    "network.harvesterhci.io",
	Version:  "v1alpha1",
	Resource: "ippools", // Make sure this matches resource name
}

type Informer struct {
	healthy  atomic.Bool
	informer cache.SharedIndexInformer
}

type IncompleteResourceError struct {
	Name  string
	Field string
}

func (e IncompleteResourceError) Error() string {
	return fmt.Sprintf("incomplete resource, field %q missing in %q", e.Field, e.Name)
}

func (in *Informer) IsHealthy() bool {
	return in.healthy.Load()
}

func (in *Informer) Run(ctx context.Context, client dynamic.Interface, resyncInterval time.Duration) {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, resyncInterval, corev1.NamespaceAll, nil)
	in.informer = factory.ForResource(Resource).Informer()

	in.informer.SetWatchErrorHandler(func(_ *cache.Reflector, _ error) {
		in.healthy.Store(false)
	})

	in.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(_ interface{}) {
			in.healthy.Store(true)
		},
		UpdateFunc: func(_, _ interface{}) {
			in.healthy.Store(true)
		},
		DeleteFunc: func(_ interface{}) {
			in.healthy.Store(true)
		},
	})

	in.informer.Run(ctx.Done())
}

func (in *Informer) GetIPPools() ([]pooldataposter.IPPool, error) {
	store := in.informer.GetStore()

	resources, err := listUnstructure(store.List())
	if err != nil {
		return nil, fmt.Errorf("failed to parse resources: %w", err)
	}
	return resources, nil
}

func listUnstructure(objects []any) ([]pooldataposter.IPPool, error) {
	resources := make([]pooldataposter.IPPool, len(objects))
	for i, obj := range objects {
		err := toIppoolRes(obj, &resources[i])
		if err != nil {
			return nil, err
		}
	}
	return resources, nil
}

func toIppoolRes(object any, pool *pooldataposter.IPPool) error {
	obj := object.(*unstructured.Unstructured)
	var ok bool

	pool.Name = obj.GetName()
	spec, ok, err := unstructured.NestedMap(obj.Object, "spec", "ipv4Config", "pool")
	if err != nil {
		return err
	} else if !ok {
		return &IncompleteResourceError{
			Name:  pool.Name,
			Field: "spec.ipv4Config.pool",
		}
	}

	start, ok, err := unstructured.NestedString(spec, "start")
	if err != nil {
		return err
	} else if !ok {
		return &IncompleteResourceError{
			Name:  pool.Name,
			Field: "spec.ipv4Config.pool.start",
		}
	}
	pool.Start = net.ParseIP(start)

	end, _, err := unstructured.NestedString(spec, "end")
	if err != nil {
		return err
	} else if !ok {
		return &IncompleteResourceError{
			Name:  pool.Name,
			Field: "spec.ipv4Config.pool.end",
		}
	}
	pool.End = net.ParseIP(end)

	return nil
}
