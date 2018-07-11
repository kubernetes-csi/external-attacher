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
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/kubernetes-csi/csi-test/driver"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	driverName = "foo/bar"
)

type pbMatcher struct {
	x proto.Message
}

func (p pbMatcher) Matches(x interface{}) bool {
	y := x.(proto.Message)
	return proto.Equal(p.x, y)
}

func (p pbMatcher) String() string {
	return fmt.Sprintf("pb equal to %v", p.x)
}

func pbMatch(x interface{}) gomock.Matcher {
	v := x.(proto.Message)
	return &pbMatcher{v}
}

func createMockServer(t *testing.T) (*gomock.Controller, *driver.MockCSIDriver, *driver.MockIdentityServer, *driver.MockControllerServer, CSIConnection, error) {
	// Start the mock server
	mockController := gomock.NewController(t)
	identityServer := driver.NewMockIdentityServer(mockController)
	controllerServer := driver.NewMockControllerServer(mockController)
	drv := driver.NewMockCSIDriver(&driver.MockCSIDriverServers{
		Identity:   identityServer,
		Controller: controllerServer,
	})
	drv.Start()

	// Create a client connection to it
	addr := drv.Address()
	csiConn, err := New(addr, 10)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return mockController, drv, identityServer, controllerServer, csiConn, nil
}

func TestGetPluginInfo(t *testing.T) {
	tests := []struct {
		name        string
		output      *csi.GetPluginInfoResponse
		injectError bool
		expectError bool
	}{
		{
			name: "success",
			output: &csi.GetPluginInfoResponse{
				Name:          "csi/example",
				VendorVersion: "0.2.0",
				Manifest: map[string]string{
					"hello": "world",
				},
			},
			expectError: false,
		},
		{
			name:        "gRPC error",
			output:      nil,
			injectError: true,
			expectError: true,
		},
		{
			name: "empty name",
			output: &csi.GetPluginInfoResponse{
				Name: "",
			},
			expectError: true,
		},
	}

	mockController, driver, identityServer, _, csiConn, err := createMockServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mockController.Finish()
	defer driver.Stop()
	defer csiConn.Close()

	for _, test := range tests {

		in := &csi.GetPluginInfoRequest{}

		out := test.output
		var injectedErr error = nil
		if test.injectError {
			injectedErr = fmt.Errorf("mock error")
		}

		// Setup expectation
		identityServer.EXPECT().GetPluginInfo(gomock.Any(), pbMatch(in)).Return(out, injectedErr).Times(1)

		name, err := csiConn.GetDriverName(context.Background())
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
		if err == nil && name != "csi/example" {
			t.Errorf("got unexpected name: %q", name)
		}
	}
}

func TestSupportsControllerPublish(t *testing.T) {
	tests := []struct {
		name        string
		output      *csi.ControllerGetCapabilitiesResponse
		injectError bool
		expectError bool
	}{
		{
			name: "success",
			output: &csi.ControllerGetCapabilitiesResponse{
				Capabilities: []*csi.ControllerServiceCapability{
					{
						Type: &csi.ControllerServiceCapability_Rpc{
							Rpc: &csi.ControllerServiceCapability_RPC{
								Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
							},
						},
					},
					{
						Type: &csi.ControllerServiceCapability_Rpc{
							Rpc: &csi.ControllerServiceCapability_RPC{
								Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "gRPC error",
			output:      nil,
			injectError: true,
			expectError: true,
		},
		{
			name: "no publish",
			output: &csi.ControllerGetCapabilitiesResponse{
				Capabilities: []*csi.ControllerServiceCapability{
					{
						Type: &csi.ControllerServiceCapability_Rpc{
							Rpc: &csi.ControllerServiceCapability_RPC{
								Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty capability",
			output: &csi.ControllerGetCapabilitiesResponse{
				Capabilities: []*csi.ControllerServiceCapability{
					{
						Type: nil,
					},
				},
			},
			expectError: false,
		},
		{
			name: "no capabilities",
			output: &csi.ControllerGetCapabilitiesResponse{
				Capabilities: []*csi.ControllerServiceCapability{},
			},
			expectError: false,
		},
	}

	mockController, driver, _, controllerServer, csiConn, err := createMockServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mockController.Finish()
	defer driver.Stop()
	defer csiConn.Close()

	for _, test := range tests {

		in := &csi.ControllerGetCapabilitiesRequest{}

		out := test.output
		var injectedErr error = nil
		if test.injectError {
			injectedErr = fmt.Errorf("mock error")
		}

		// Setup expectation
		controllerServer.EXPECT().ControllerGetCapabilities(gomock.Any(), pbMatch(in)).Return(out, injectedErr).Times(1)

		_, err = csiConn.SupportsControllerPublish(context.Background())
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
	}
}

func TestSupportsPluginControllerService(t *testing.T) {
	tests := []struct {
		name        string
		output      *csi.GetPluginCapabilitiesResponse
		injectError bool
		expectError bool
	}{
		{
			name: "success",
			output: &csi.GetPluginCapabilitiesResponse{
				Capabilities: []*csi.PluginCapability{
					{
						Type: &csi.PluginCapability_Service_{
							Service: &csi.PluginCapability_Service{
								Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
							},
						},
					},
					{
						Type: &csi.PluginCapability_Service_{
							Service: &csi.PluginCapability_Service{
								Type: csi.PluginCapability_Service_UNKNOWN,
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "gRPC error",
			output:      nil,
			injectError: true,
			expectError: true,
		},
		{
			name: "no controller service",
			output: &csi.GetPluginCapabilitiesResponse{
				Capabilities: []*csi.PluginCapability{
					{
						Type: &csi.PluginCapability_Service_{
							Service: &csi.PluginCapability_Service{
								Type: csi.PluginCapability_Service_UNKNOWN,
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty capability",
			output: &csi.GetPluginCapabilitiesResponse{
				Capabilities: []*csi.PluginCapability{
					{
						Type: nil,
					},
				},
			},
			expectError: false,
		},
		{
			name: "no capabilities",
			output: &csi.GetPluginCapabilitiesResponse{
				Capabilities: []*csi.PluginCapability{},
			},
			expectError: false,
		},
	}

	mockController, driver, identityServer, _, csiConn, err := createMockServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mockController.Finish()
	defer driver.Stop()
	defer csiConn.Close()

	for _, test := range tests {

		in := &csi.GetPluginCapabilitiesRequest{}

		out := test.output
		var injectedErr error = nil
		if test.injectError {
			injectedErr = fmt.Errorf("mock error")
		}

		// Setup expectation
		identityServer.EXPECT().GetPluginCapabilities(gomock.Any(), pbMatch(in)).Return(out, injectedErr).Times(1)

		_, err = csiConn.SupportsPluginControllerService(context.Background())
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
	}
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
		VolumeAttributes: map[string]string{"foo": "bar"},
		Readonly:         false,
	}
	secretsRequest := &csi.ControllerPublishVolumeRequest{
		VolumeId:                 defaultVolumeID,
		NodeId:                   defaultNodeID,
		VolumeCapability:         defaultCaps,
		ControllerPublishSecrets: map[string]string{"foo": "bar"},
		Readonly:                 false,
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
				PublishInfo: publishVolumeInfo,
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
				PublishInfo: publishVolumeInfo,
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
				PublishInfo: publishVolumeInfo,
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
				PublishInfo: publishVolumeInfo,
			},
			expectError:    false,
			expectedInfo:   publishVolumeInfo,
			expectDetached: false,
		},
	}

	mockController, driver, _, controllerServer, csiConn, err := createMockServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mockController.Finish()
	defer driver.Stop()
	defer csiConn.Close()

	for _, test := range tests {
		in := test.input
		out := test.output
		var injectedErr error = nil
		if test.injectError != codes.OK {
			injectedErr = status.Error(test.injectError, fmt.Sprintf("Injecting error %d", test.injectError))
		}

		// Setup expectation
		if in != nil {
			controllerServer.EXPECT().ControllerPublishVolume(gomock.Any(), pbMatch(in)).Return(out, injectedErr).Times(1)
		}

		publishInfo, detached, err := csiConn.Attach(context.Background(), test.volumeID, test.readonly, test.nodeID, test.caps, test.attributes, test.secrets)
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
		if err == nil && !reflect.DeepEqual(publishInfo, test.expectedInfo) {
			t.Errorf("got unexpected PublishInfo: %+v", publishInfo)
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
		ControllerUnpublishSecrets: map[string]string{"foo": "bar"},
	}

	tests := []struct {
		name           string
		volumeID       string
		nodeID         string
		secrets        map[string]string
		input          *csi.ControllerUnpublishVolumeRequest
		output         *csi.ControllerUnpublishVolumeResponse
		injectError    codes.Code
		expectError    bool
		expectDetached bool
	}{
		{
			name:           "success",
			volumeID:       defaultVolumeID,
			nodeID:         defaultNodeID,
			input:          defaultRequest,
			output:         &csi.ControllerUnpublishVolumeResponse{},
			expectError:    false,
			expectDetached: true,
		},
		{
			name:           "secrets",
			volumeID:       defaultVolumeID,
			nodeID:         defaultNodeID,
			secrets:        map[string]string{"foo": "bar"},
			input:          secretsRequest,
			output:         &csi.ControllerUnpublishVolumeResponse{},
			expectError:    false,
			expectDetached: true,
		},
		{
			name:           "gRPC final error",
			volumeID:       defaultVolumeID,
			nodeID:         defaultNodeID,
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
			input:          defaultRequest,
			output:         nil,
			injectError:    codes.DeadlineExceeded,
			expectError:    true,
			expectDetached: false,
		},
	}

	mockController, driver, _, controllerServer, csiConn, err := createMockServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer mockController.Finish()
	defer driver.Stop()
	defer csiConn.Close()

	for _, test := range tests {
		in := test.input
		out := test.output
		var injectedErr error = nil
		if test.injectError != codes.OK {
			injectedErr = status.Error(test.injectError, fmt.Sprintf("Injecting error %d", test.injectError))
		}

		// Setup expectation
		if in != nil {
			controllerServer.EXPECT().ControllerUnpublishVolume(gomock.Any(), pbMatch(in)).Return(out, injectedErr).Times(1)
		}

		detached, err := csiConn.Detach(context.Background(), test.volumeID, test.nodeID, test.secrets)
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
		if detached != test.expectDetached {
			t.Errorf("test %q: expected detached=%v, got %v", test.name, test.expectDetached, detached)
		}
	}
}
