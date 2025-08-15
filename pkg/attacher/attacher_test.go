/*
Copyright 2019 The Kubernetes Authors.

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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/metrics"
	"github.com/kubernetes-csi/csi-test/v5/driver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type pbMatcher struct {
	x proto.Message
}

func (p pbMatcher) Matches(x any) bool {
	y := x.(proto.Message)
	return proto.Equal(p.x, y)
}

func (p pbMatcher) String() string {
	return fmt.Sprintf("pb equal to %v", p.x)
}

func pbMatch(x any) gomock.Matcher {
	v := x.(proto.Message)
	return &pbMatcher{v}
}

func tempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "external-attacher-test-")
	if err != nil {
		t.Fatalf("Cannot create temporary directory: %s", err)
	}
	return dir
}

func createMockServer(t *testing.T, tmpdir string) (*gomock.Controller, *driver.MockCSIDriver, *driver.MockIdentityServer, *driver.MockControllerServer, *grpc.ClientConn, error) {
	// Start the mock server
	mockController := gomock.NewController(t)
	identityServer := driver.NewMockIdentityServer(mockController)
	controllerServer := driver.NewMockControllerServer(mockController)
	metricsManager := metrics.NewCSIMetricsManager("test.csi.driver.io" /* driverName */)
	drv := driver.NewMockCSIDriver(&driver.MockCSIDriverServers{
		Identity:   identityServer,
		Controller: controllerServer,
	})
	drv.StartOnAddress("unix", filepath.Join(tmpdir, "csi.sock"))

	// Create a client connection to it
	addr := drv.Address()
	t.Logf("adds: %s", addr)
	ctx := context.Background()
	csiConn, err := connection.Connect(ctx, addr, metricsManager)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return mockController, drv, identityServer, controllerServer, csiConn, nil
}

func TestAttach(t *testing.T) {
	defaultVolumeID := "myname"
	defaultNodeID := "MyNodeID"
	defaultCaps := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
		},
	}
	publishVolumeInfo := map[string]string{
		"first":  "foo",
		"second": "bar",
		"third":  "baz",
	}
	defaultRequest := &csi.ControllerPublishVolumeRequest{
		VolumeId:         defaultVolumeID,
		NodeId:           defaultNodeID,
		VolumeCapability: defaultCaps,
		Readonly:         false,
	}
	readOnlyRequest := &csi.ControllerPublishVolumeRequest{
		VolumeId:         defaultVolumeID,
		NodeId:           defaultNodeID,
		VolumeCapability: defaultCaps,
		Readonly:         true,
	}
	attributesRequest := &csi.ControllerPublishVolumeRequest{
		VolumeId:         defaultVolumeID,
		NodeId:           defaultNodeID,
		VolumeCapability: defaultCaps,
		VolumeContext:    map[string]string{"foo": "bar"},
		Readonly:         false,
	}
	secretsRequest := &csi.ControllerPublishVolumeRequest{
		VolumeId:         defaultVolumeID,
		NodeId:           defaultNodeID,
		VolumeCapability: defaultCaps,
		Secrets:          map[string]string{"foo": "bar"},
		Readonly:         false,
	}

	tests := []struct {
		name           string
		volumeID       string
		nodeID         string
		caps           *csi.VolumeCapability
		readonly       bool
		attributes     map[string]string
		secrets        map[string]string
		input          *csi.ControllerPublishVolumeRequest
		output         *csi.ControllerPublishVolumeResponse
		injectError    codes.Code
		expectError    bool
		expectDetached bool
		expectedInfo   map[string]string
	}{
		{
			name:     "success",
			volumeID: defaultVolumeID,
			nodeID:   defaultNodeID,
			caps:     defaultCaps,
			input:    defaultRequest,
			output: &csi.ControllerPublishVolumeResponse{
				PublishContext: publishVolumeInfo,
			},
			expectError:    false,
			expectedInfo:   publishVolumeInfo,
			expectDetached: false,
		},
		{
			name:           "success no info",
			volumeID:       defaultVolumeID,
			nodeID:         defaultNodeID,
			caps:           defaultCaps,
			input:          defaultRequest,
			output:         &csi.ControllerPublishVolumeResponse{},
			expectError:    false,
			expectedInfo:   nil,
			expectDetached: false,
		},
		{
			name:     "readonly success",
			volumeID: defaultVolumeID,
			nodeID:   defaultNodeID,
			caps:     defaultCaps,
			readonly: true,
			input:    readOnlyRequest,
			output: &csi.ControllerPublishVolumeResponse{
				PublishContext: publishVolumeInfo,
			},
			expectError:    false,
			expectedInfo:   publishVolumeInfo,
			expectDetached: false,
		},
		{
			name:           "gRPC final error",
			volumeID:       defaultVolumeID,
			nodeID:         defaultNodeID,
			caps:           defaultCaps,
			input:          defaultRequest,
			output:         nil,
			injectError:    codes.NotFound,
			expectError:    true,
			expectDetached: true,
		},
		{
			name:           "gRPC transient error",
			volumeID:       defaultVolumeID,
			nodeID:         defaultNodeID,
			caps:           defaultCaps,
			input:          defaultRequest,
			output:         nil,
			injectError:    codes.DeadlineExceeded,
			expectError:    true,
			expectDetached: false,
		},
		{
			name:       "attributes",
			volumeID:   defaultVolumeID,
			nodeID:     defaultNodeID,
			caps:       defaultCaps,
			attributes: map[string]string{"foo": "bar"},
			input:      attributesRequest,
			output: &csi.ControllerPublishVolumeResponse{
				PublishContext: publishVolumeInfo,
			},
			expectError:    false,
			expectedInfo:   publishVolumeInfo,
			expectDetached: false,
		},
		{
			name:     "secrets",
			volumeID: defaultVolumeID,
			nodeID:   defaultNodeID,
			caps:     defaultCaps,
			secrets:  map[string]string{"foo": "bar"},
			input:    secretsRequest,
			output: &csi.ControllerPublishVolumeResponse{
				PublishContext: publishVolumeInfo,
			},
			expectError:    false,
			expectedInfo:   publishVolumeInfo,
			expectDetached: false,
		},
	}

	tmpdir := tempDir(t)
	defer os.RemoveAll(tmpdir)
	mockController, driver, _, controllerServer, csiConn, err := createMockServer(t, tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	defer mockController.Finish()
	defer driver.Stop()

	for _, test := range tests {
		in := test.input
		out := test.output
		var injectedErr error
		if test.injectError != codes.OK {
			injectedErr = status.Error(test.injectError, fmt.Sprintf("Injecting error %d", test.injectError))
		}

		// Setup expectation
		if in != nil {
			controllerServer.EXPECT().ControllerPublishVolume(gomock.Any(), pbMatch(in)).Return(out, injectedErr).Times(1)
		}

		a := NewAttacher(csiConn)
		publishInfo, detached, err := a.Attach(context.Background(), test.volumeID, test.readonly, test.nodeID, test.caps, test.attributes, test.secrets)
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
		if err == nil && !reflect.DeepEqual(publishInfo, test.expectedInfo) {
			t.Errorf("got unexpected PublishContext: %+v", publishInfo)
		}
		if detached != test.expectDetached {
			t.Errorf("test %q: expected detached=%v, got %v", test.name, test.expectDetached, detached)
		}
	}
}

func TestDetachAttach(t *testing.T) {
	defaultVolumeID := "myname"

	defaultNodeID := "MyNodeID"

	defaultRequest := &csi.ControllerUnpublishVolumeRequest{
		VolumeId: defaultVolumeID,
		NodeId:   defaultNodeID,
	}

	secretsRequest := &csi.ControllerUnpublishVolumeRequest{
		VolumeId: defaultVolumeID,
		NodeId:   defaultNodeID,
		Secrets:  map[string]string{"foo": "bar"},
	}

	tests := []struct {
		name        string
		volumeID    string
		nodeID      string
		secrets     map[string]string
		input       *csi.ControllerUnpublishVolumeRequest
		output      *csi.ControllerUnpublishVolumeResponse
		injectError codes.Code
		expectError bool
	}{
		{
			name:        "success",
			volumeID:    defaultVolumeID,
			nodeID:      defaultNodeID,
			input:       defaultRequest,
			output:      &csi.ControllerUnpublishVolumeResponse{},
			expectError: false,
		},
		{
			name:        "secrets",
			volumeID:    defaultVolumeID,
			nodeID:      defaultNodeID,
			secrets:     map[string]string{"foo": "bar"},
			input:       secretsRequest,
			output:      &csi.ControllerUnpublishVolumeResponse{},
			expectError: false,
		},
		{
			name:        "gRPC final error",
			volumeID:    defaultVolumeID,
			nodeID:      defaultNodeID,
			input:       defaultRequest,
			output:      nil,
			injectError: codes.PermissionDenied,
			expectError: true,
		},
		{
			name:        "gRPC transient error",
			volumeID:    defaultVolumeID,
			nodeID:      defaultNodeID,
			input:       defaultRequest,
			output:      nil,
			injectError: codes.DeadlineExceeded,
			expectError: true,
		},
		{
			// Explicitly test NotFound, it's handled as any other error.
			// https://github.com/kubernetes-csi/external-attacher/pull/165
			name:        "gRPC NotFound error",
			volumeID:    defaultVolumeID,
			nodeID:      defaultNodeID,
			input:       defaultRequest,
			output:      nil,
			injectError: codes.NotFound,
			expectError: true,
		},
	}

	tmpdir := tempDir(t)
	defer os.RemoveAll(tmpdir)
	mockController, driver, _, controllerServer, csiConn, err := createMockServer(t, tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	defer mockController.Finish()
	defer driver.Stop()

	for _, test := range tests {
		in := test.input
		out := test.output
		var injectedErr error
		if test.injectError != codes.OK {
			injectedErr = status.Error(test.injectError, fmt.Sprintf("Injecting error %d", test.injectError))
		}

		// Setup expectation
		if in != nil {
			controllerServer.EXPECT().ControllerUnpublishVolume(gomock.Any(), pbMatch(in)).Return(out, injectedErr).Times(1)
		}

		a := NewAttacher(csiConn)
		err := a.Detach(context.Background(), test.volumeID, test.nodeID, test.secrets)
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
	}
}
