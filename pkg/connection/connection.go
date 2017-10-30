/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connection

import (
	"context"
	"fmt"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// CSIConnection is gRPC connection to a remote CSI driver and abstracts all
// CSI calls.
type CSIConnection interface {
	// GetDriverName returns driver name as discovered by GetPluginInfo()
	// gRPC call.
	GetDriverName(ctx context.Context) (string, error)

	// SupportsControllerPublish returns true if the CSI driver reports
	// PUBLISH_UNPUBLISH_VOLUME in ControllerGetCapabilities() gRPC call.
	SupportsControllerPublish(ctx context.Context) (bool, error)

	// Close the connection
	Close() error
}

type csiConnection struct {
	conn *grpc.ClientConn
}

var (
	_ CSIConnection = &csiConnection{}

	// Version of CSI this client implements
	csiVersion = csi.Version{
		Major: 0,
		Minor: 1,
		Patch: 0,
	}
)

func New(address string, timeout time.Duration) (CSIConnection, error) {
	conn, err := connect(address, timeout)
	if err != nil {
		return nil, err
	}
	return &csiConnection{
		conn: conn,
	}, nil
}

func connect(address string, timeout time.Duration) (*grpc.ClientConn, error) {
	glog.V(2).Infof("Connecting to %s", address)
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBackoffMaxDelay(time.Second))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		if !conn.WaitForStateChange(ctx, conn.GetState()) {
			glog.V(4).Infof("Connection timed out")
			return conn, nil // return nil, subsequent GetPluginInfo will show the real connection error
		}
		if conn.GetState() == connectivity.Ready {
			glog.V(3).Infof("Connected")
			return conn, nil
		}
		glog.V(4).Infof("Still trying, connection is %s", conn.GetState())
	}
}

func (c *csiConnection) GetDriverName(ctx context.Context) (string, error) {
	client := csi.NewIdentityClient(c.conn)

	req := csi.GetPluginInfoRequest{
		Version: &csiVersion,
	}

	rsp, err := client.GetPluginInfo(ctx, &req)
	if err != nil {
		return "", err
	}
	e := rsp.GetError()
	if e != nil {
		// TODO: report the right error
		return "", fmt.Errorf("Error calling GetPluginInfo: %+v", e)
	}

	result := rsp.GetResult()
	if result == nil {
		return "", fmt.Errorf("result is empty")
	}
	name := result.GetName()
	if name == "" {
		return "", fmt.Errorf("name is empty")
	}
	return result.GetName(), nil
}

func (c *csiConnection) SupportsControllerPublish(ctx context.Context) (bool, error) {
	client := csi.NewControllerClient(c.conn)
	req := csi.ControllerGetCapabilitiesRequest{
		Version: &csiVersion,
	}

	rsp, err := client.ControllerGetCapabilities(ctx, &req)
	if err != nil {
		return false, err
	}
	e := rsp.GetError()
	if e != nil {
		// TODO: report the right error
		return false, fmt.Errorf("error calling ControllerGetCapabilities: %+v", e)
	}

	result := rsp.GetResult()
	if result == nil {
		return false, fmt.Errorf("result is empty")
	}

	caps := result.GetCapabilities()
	for _, cap := range caps {
		if cap == nil {
			continue
		}
		rpc := cap.GetRpc()
		if rpc == nil {
			continue
		}
		if rpc.GetType() == csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME {
			return true, nil
		}
	}
	return false, nil
}

func (c *csiConnection) Close() error {
	return c.conn.Close()
}
