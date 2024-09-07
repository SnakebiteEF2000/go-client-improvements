package main

import "k8s.io/apimachinery/pkg/runtime/schema"

//should load that from env or whatever

var Resource = schema.GroupVersionResource{
	Group:    "network.harvesterhci.io",
	Version:  "v1alpha1",
	Resource: "ippools", // Make sure this matches resource name
}
