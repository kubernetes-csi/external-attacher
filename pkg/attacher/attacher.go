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

package attacher

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Attacher implements attach/detach operations against a remote CSI driver.
type Attacher interface {
	// Attach given volume to given node. Returns PublishVolumeInfo. Note that
	// "detached" is returned on error and means that the volume is for sure
	// detached from the node. "false" means that the volume may be either
	// detached, attaching or attached and caller should retry to get the final
	// status.
	Attach(ctx context.Context, volumeID string, readOnly bool, nodeID string, caps *csi.VolumeCapability, attributes, secrets map[string]string) (metadata map[string]string, detached bool, err error)

	// Detach given volume from given node.
	Detach(ctx context.Context, volumeID string, nodeID string, secrets map[string]string) error
}

type attacher struct {
	client csi.ControllerClient
}

var (
	_ Attacher = &attacher{}
)

// NewAttacher provides a new Attacher object.
func NewAttacher(conn *grpc.ClientConn) Attacher {
	return &attacher{
		client: csi.NewControllerClient(conn),
	}
}

func (a *attacher) Attach(ctx context.Context, volumeID string, readOnly bool, nodeID string, caps *csi.VolumeCapability, context, secrets map[string]string) (metadata map[string]string, detached bool, err error) {
	req := csi.ControllerPublishVolumeRequest{
		VolumeId:         volumeID,
		NodeId:           nodeID,
		VolumeCapability: caps,
		Readonly:         readOnly,
		VolumeContext:    context,
		Secrets:          secrets,
	}

	rsp, err := a.client.ControllerPublishVolume(ctx, &req)
	if err != nil {
		return nil, isFinalError(err), err
	}
	return rsp.PublishContext, false, nil
}

func (a *attacher) Detach(ctx context.Context, volumeID string, nodeID string, secrets map[string]string) error {
	req := csi.ControllerUnpublishVolumeRequest{
		VolumeId: volumeID,
		NodeId:   nodeID,
		Secrets:  secrets,
	}

	_, err := a.client.ControllerUnpublishVolume(ctx, &req)
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
