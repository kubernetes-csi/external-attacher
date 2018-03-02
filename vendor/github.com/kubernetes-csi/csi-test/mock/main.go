package main

import (
	"context"

	"github.com/kubernetes-csi/csi-test/mock/gocsi"
	"github.com/kubernetes-csi/csi-test/mock/provider"
	"github.com/kubernetes-csi/csi-test/mock/service"
)

// main is ignored when this package is built as a go plug-in
func main() {
	gocsi.Run(
		context.Background(),
		service.Name,
		"A Mock Container Storage Interface (CSI) Storage Plug-in (SP)",
		"",
		provider.New())
}
