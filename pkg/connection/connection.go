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
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

// CSIConnection is gRPC connection to a remote CSI driver and abstracts all
// CSI calls.
type CSIConnection interface {
	// GetDriverName returns driver name as discovered by GetPluginInfo()
	// gRPC call.
	GetDriverName(ctx context.Context) (string, error)

	// SupportsControllerPublish returns true if the CSI driver reports
	// PUBLISH_UNPUBLISH_VOLUME in ControllerGetCapabilities() gRPC call.
	SupportsControllerPublish(ctx context.Context) (supportsControllerPublish bool, supportsPublishReadOnly bool, err error)

	// SupportsPluginControllerService return true if the CSI driver reports
	// CONTROLLER_SERVICE in GetPluginCapabilities() gRPC call.
	SupportsPluginControllerService(ctx context.Context) (bool, error)

	// Attach given volume to given node. Returns PublishVolumeInfo. Note that
	// "detached" is returned on error and means that the volume is for sure
	// detached from the node. "false" means that the volume may be either
	// detached, attaching or attached and caller should retry to get the final
	// status.
	Attach(ctx context.Context, volumeID string, readOnly bool, nodeID string, caps *csi.VolumeCapability, attributes, secrets map[string]string) (metadata map[string]string, detached bool, err error)

	// Detach given volume from given node. Note that "detached" is returned on
	// error and means that the volume is for sure detached from the node.
	// "false" means that the volume may or may not be detached and caller
	// should retry.
	Detach(ctx context.Context, volumeID string, nodeID string, secrets map[string]string) (detached bool, err error)

	// Probe checks that the CSI driver is ready to process requests
	Probe(singleProbeTimeout time.Duration) error

	// Close the connection
	Close() error
}

type csiConnection struct {
	conn         *grpc.ClientConn
	capabilities []csi.ControllerServiceCapability
}

var (
	_ CSIConnection = &csiConnection{}
)

// New provides a new CSI connection object.
func New(address string) (CSIConnection, error) {
	conn, err := connection.Connect(address, connection.OnConnectionLoss(connection.ExitOnConnectionLoss()))
	if err != nil {
		return nil, err
	}
	return &csiConnection{
		conn: conn,
	}, nil
}

func (c *csiConnection) GetDriverName(ctx context.Context) (string, error) {
	return connection.GetDriverName(ctx, c.conn)
}

func (c *csiConnection) Probe(singleProbeTimeout time.Duration) error {
	return connection.ProbeForever(c.conn, singleProbeTimeout)
}

func (c *csiConnection) SupportsControllerPublish(ctx context.Context) (supportsControllerPublish bool, supportsPublishReadOnly bool, err error) {
	caps, err := connection.GetControllerCapabilities(ctx, c.conn)
	if err != nil {
		return false, false, err
	}

	supportsControllerPublish = caps[csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME]
	supportsPublishReadOnly = caps[csi.ControllerServiceCapability_RPC_PUBLISH_READONLY]
	return supportsControllerPublish, supportsPublishReadOnly, nil
}

func (c *csiConnection) SupportsPluginControllerService(ctx context.Context) (bool, error) {
	caps, err := connection.GetPluginCapabilities(ctx, c.conn)
	if err != nil {
		return false, err
	}

	return caps[csi.PluginCapability_Service_CONTROLLER_SERVICE], nil
}

func (c *csiConnection) Attach(ctx context.Context, volumeID string, readOnly bool, nodeID string, caps *csi.VolumeCapability, context, secrets map[string]string) (metadata map[string]string, detached bool, err error) {
	client := csi.NewControllerClient(c.conn)

	req := csi.ControllerPublishVolumeRequest{
		VolumeId:         volumeID,
		NodeId:           nodeID,
		VolumeCapability: caps,
		Readonly:         readOnly,
		VolumeContext:    context,
		Secrets:          secrets,
	}

	rsp, err := client.ControllerPublishVolume(ctx, &req)
	if err != nil {
		return nil, isFinalError(err), err
	}
	return rsp.PublishContext, false, nil
}

func (c *csiConnection) Detach(ctx context.Context, volumeID string, nodeID string, secrets map[string]string) (detached bool, err error) {
	client := csi.NewControllerClient(c.conn)

	req := csi.ControllerUnpublishVolumeRequest{
		VolumeId: volumeID,
		NodeId:   nodeID,
		Secrets:  secrets,
	}

	_, err = client.ControllerUnpublishVolume(ctx, &req)
	if err != nil {
		return isFinalError(err), err
	}
	return true, nil
}

func (c *csiConnection) Close() error {
	return c.conn.Close()
}

func logGRPC(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	klog.V(5).Infof("GRPC call: %s", method)
	klog.V(5).Infof("GRPC request: %s", protosanitizer.StripSecrets(req))
	err := invoker(ctx, method, req, reply, cc, opts...)
	klog.V(5).Infof("GRPC response: %s", protosanitizer.StripSecrets(reply))
	klog.V(5).Infof("GRPC error: %v", err)
	return err
}

// isFinished returns true if given error represents final error of an
// operation. That means the operation has failed completely and cannot be in
// progress.  It returns false, if the error represents some transient error
// like timeout and the operation itself or previous call to the same
// operation can be actually in progress.
func isFinalError(err error) bool {
	// Sources:
	// https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
	// https://github.com/container-storage-interface/spec/blob/master/spec.md
	st, ok := status.FromError(err)
	if !ok {
		// This is not gRPC error. The operation must have failed before gRPC
		// method was called, otherwise we would get gRPC error.
		return false
	}
	switch st.Code() {
	case codes.Canceled, // gRPC: Client Application cancelled the request
		codes.DeadlineExceeded,  // gRPC: Timeout
		codes.Unavailable,       // gRPC: Server shutting down, TCP connection broken - previous Attach() or Detach() may be still in progress.
		codes.ResourceExhausted, // gRPC: Server temporarily out of resources - previous Attach() or Detach() may be still in progress.
		codes.Aborted:           // CSI: Operation pending for volume
		return false
	}
	// All other errors mean that the operation (attach/detach) either did not
	// even start or failed. It is for sure not in progress.
	return true
}
