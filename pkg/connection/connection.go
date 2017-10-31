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
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"k8s.io/api/core/v1"
)

const (
	nodeIDAnnotation = "nodeid.csi.volume.kubernetes.io/"

	// Key for node name in NodeID
	nodeNameKey = "Name"
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

	// Attach given volume to given node. Returns PublishVolumeInfo
	Attach(ctx context.Context, pv *v1.PersistentVolume, node *v1.Node) (map[string]string, error)

	// Detach given volume from given node.
	Detach(ctx context.Context, pv *v1.PersistentVolume, node *v1.Node) error

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

func (c *csiConnection) Attach(ctx context.Context, pv *v1.PersistentVolume, node *v1.Node) (map[string]string, error) {
	client := csi.NewControllerClient(c.conn)

	if pv.Spec.CSI == nil {
		return nil, fmt.Errorf("only CSI volumes are supported")
	}

	nodeID, err := getNodeID(pv.Spec.CSI.Driver, node)
	if err != nil {
		return nil, err
	}

	caps, err := getVolumeCapabilities(pv)
	if err != nil {
		return nil, err
	}

	req := csi.ControllerPublishVolumeRequest{
		Version: &csiVersion,
		VolumeHandle: &csi.VolumeHandle{
			Id: pv.Spec.CSI.VolumeHandle,
			// TODO: add metadata???
		},
		NodeId:           nodeID,
		VolumeCapability: caps,
		Readonly:         pv.Spec.CSI.ReadOnly,
		UserCredentials:  nil,
	}

	rsp, err := client.ControllerPublishVolume(ctx, &req)
	if err != nil {
		return nil, err
	}
	e := rsp.GetError()
	if e != nil {
		// TODO: report the right error
		return nil, fmt.Errorf("error calling ControllerPublishVolume: %+v", e)
	}

	result := rsp.GetResult()
	if result == nil {
		return nil, fmt.Errorf("result is empty")
	}

	return result.PublishVolumeInfo, nil
}

func (c *csiConnection) Detach(ctx context.Context, pv *v1.PersistentVolume, node *v1.Node) error {
	client := csi.NewControllerClient(c.conn)

	if pv.Spec.CSI == nil {
		return fmt.Errorf("only CSI volumes are supported")
	}

	nodeID, err := getNodeID(pv.Spec.CSI.Driver, node)
	if err != nil {
		return err
	}

	req := csi.ControllerUnpublishVolumeRequest{
		Version: &csiVersion,
		VolumeHandle: &csi.VolumeHandle{
			Id: pv.Spec.CSI.VolumeHandle,
			// TODO: add metadata???
		},
		NodeId:          nodeID,
		UserCredentials: nil,
	}

	rsp, err := client.ControllerUnpublishVolume(ctx, &req)
	if err != nil {
		return err
	}
	e := rsp.GetError()
	if e != nil {
		// TODO: report the right error
		return fmt.Errorf("error calling ControllerUnpublishVolume: %+v", e)
	}

	result := rsp.GetResult()
	if result == nil {
		return fmt.Errorf("result is empty")
	}

	return nil
}

func sanitizeDriverName(driver string) string {
	// replace '/' with '_'
	return strings.Replace(driver, "/", "_", -1)
}

func getNodeID(driver string, node *v1.Node) (*csi.NodeID, error) {
	annotationName := nodeIDAnnotation + sanitizeDriverName(driver)
	nodeID, ok := node.Annotations[annotationName]
	if !ok {
		return nil, fmt.Errorf("node %q has no NodeID for driver %q", node.Name, driver)
	}
	return &csi.NodeID{
		Values: map[string]string{
			// TODO: find out what key is expected.
			nodeNameKey: nodeID,
		},
	}, nil
}

func getVolumeCapabilities(pv *v1.PersistentVolume) (*csi.VolumeCapability, error) {
	m := map[v1.PersistentVolumeAccessMode]bool{}
	for _, mode := range pv.Spec.AccessModes {
		m[mode] = true
	}

	cap := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				// TODO: get FsType from somewhere
				MountFlags: pv.Spec.MountOptions,
			},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{},
	}

	// Translate array of modes into single VolumeCapability
	switch {
	case m[v1.ReadWriteMany]:
		// ReadWriteMany trumps everything, regardless what other modes are set
		cap.AccessMode.Mode = csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER

	case m[v1.ReadOnlyMany] && m[v1.ReadWriteOnce]:
		// This is no way how to translate this to CSI...
		return nil, fmt.Errorf("CSI does not support ReadOnlyMany and ReadWriteOnce on the same PersistentVolume")

	case m[v1.ReadOnlyMany]:
		// There is only ReadOnlyMany set
		cap.AccessMode.Mode = csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY

	case m[v1.ReadWriteOnce]:
		// There is only ReadWriteOnce set
		cap.AccessMode.Mode = csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER

	default:
		return nil, fmt.Errorf("unsupported AccessMode combination: %+v", pv.Spec.AccessModes)
	}
	return cap, nil
}

func (c *csiConnection) Close() error {
	return c.conn.Close()
}
