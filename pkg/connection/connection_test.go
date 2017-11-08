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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/mock/gomock"
	"github.com/kubernetes-csi/csi-test/driver"
)

const (
	driverName = "foo/bar"
)

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
				Reply: &csi.GetPluginInfoResponse_Result_{
					Result: &csi.GetPluginInfoResponse_Result{
						Name:          "csi/example",
						VendorVersion: "0.1.0",
						Manifest: map[string]string{
							"hello": "world",
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
			name: "empty reply",
			output: &csi.GetPluginInfoResponse{
				Reply: nil,
			},
			expectError: true,
		},
		{
			name: "empty name",
			output: &csi.GetPluginInfoResponse{
				Reply: &csi.GetPluginInfoResponse_Result_{
					Result: &csi.GetPluginInfoResponse_Result{
						Name: "",
					},
				},
			},
			expectError: true,
		},
		{
			name: "general error",
			output: &csi.GetPluginInfoResponse{
				Reply: &csi.GetPluginInfoResponse_Error{
					Error: &csi.Error{
						Value: &csi.Error_GeneralError_{
							GeneralError: &csi.Error_GeneralError{
								ErrorCode:          csi.Error_GeneralError_UNSUPPORTED_REQUEST_VERSION,
								CallerMustNotRetry: true,
								ErrorDescription:   "mock error 1",
							},
						},
					},
				},
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

		in := &csi.GetPluginInfoRequest{
			Version: &csi.Version{
				Major: 0,
				Minor: 1,
				Patch: 0,
			},
		}

		out := test.output
		var injectedErr error = nil
		if test.injectError {
			injectedErr = fmt.Errorf("mock error")
		}

		// Setup expectation
		identityServer.EXPECT().GetPluginInfo(gomock.Any(), in).Return(out, injectedErr).Times(1)

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
				Reply: &csi.ControllerGetCapabilitiesResponse_Result_{
					Result: &csi.ControllerGetCapabilitiesResponse_Result{
						Capabilities: []*csi.ControllerServiceCapability{
							&csi.ControllerServiceCapability{
								Type: &csi.ControllerServiceCapability_Rpc{
									Rpc: &csi.ControllerServiceCapability_RPC{
										Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
									},
								},
							},
							&csi.ControllerServiceCapability{
								Type: &csi.ControllerServiceCapability_Rpc{
									Rpc: &csi.ControllerServiceCapability_RPC{
										Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
									},
								},
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
			name: "empty reply",
			output: &csi.ControllerGetCapabilitiesResponse{
				Reply: nil,
			},
			expectError: true,
		},
		{
			name: "general error",
			output: &csi.ControllerGetCapabilitiesResponse{
				Reply: &csi.ControllerGetCapabilitiesResponse_Error{
					Error: &csi.Error{
						Value: &csi.Error_GeneralError_{
							GeneralError: &csi.Error_GeneralError{
								ErrorCode:          csi.Error_GeneralError_UNSUPPORTED_REQUEST_VERSION,
								CallerMustNotRetry: true,
								ErrorDescription:   "mock error 1",
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "no publish",
			output: &csi.ControllerGetCapabilitiesResponse{
				Reply: &csi.ControllerGetCapabilitiesResponse_Result_{
					Result: &csi.ControllerGetCapabilitiesResponse_Result{
						Capabilities: []*csi.ControllerServiceCapability{
							&csi.ControllerServiceCapability{
								Type: &csi.ControllerServiceCapability_Rpc{
									Rpc: &csi.ControllerServiceCapability_RPC{
										Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
									},
								},
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
				Reply: &csi.ControllerGetCapabilitiesResponse_Result_{
					Result: &csi.ControllerGetCapabilitiesResponse_Result{
						Capabilities: []*csi.ControllerServiceCapability{
							&csi.ControllerServiceCapability{
								Type: nil,
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "no capabilities",
			output: &csi.ControllerGetCapabilitiesResponse{
				Reply: &csi.ControllerGetCapabilitiesResponse_Result_{
					Result: &csi.ControllerGetCapabilitiesResponse_Result{
						Capabilities: []*csi.ControllerServiceCapability{},
					},
				},
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

		in := &csi.ControllerGetCapabilitiesRequest{
			Version: &csi.Version{
				Major: 0,
				Minor: 1,
				Patch: 0,
			},
		}

		out := test.output
		var injectedErr error = nil
		if test.injectError {
			injectedErr = fmt.Errorf("mock error")
		}

		// Setup expectation
		controllerServer.EXPECT().ControllerGetCapabilities(gomock.Any(), in).Return(out, injectedErr).Times(1)

		_, err = csiConn.SupportsControllerPublish(context.Background())
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
	}
}

func TestAttach(t *testing.T) {
	defaultVolumeName := "myname"
	defaultHandle := &csi.VolumeHandle{
		Id: defaultVolumeName,
	}
	defaultNodeID := &csi.NodeID{Values: map[string]string{"Name": "MyNodeID"}}
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
		Version: &csiVersion,
		VolumeHandle: &csi.VolumeHandle{
			Id: defaultVolumeName,
		},
		NodeId:           defaultNodeID,
		VolumeCapability: defaultCaps,
		Readonly:         false,
	}
	readOnlyRequest := &csi.ControllerPublishVolumeRequest{
		Version: &csiVersion,
		VolumeHandle: &csi.VolumeHandle{
			Id: defaultVolumeName,
		},
		NodeId:           defaultNodeID,
		VolumeCapability: defaultCaps,
		Readonly:         true,
	}

	tests := []struct {
		name         string
		handle       *csi.VolumeHandle
		nodeID       *csi.NodeID
		caps         *csi.VolumeCapability
		readonly     bool
		input        *csi.ControllerPublishVolumeRequest
		output       *csi.ControllerPublishVolumeResponse
		injectError  bool
		expectError  bool
		expectedInfo map[string]string
	}{
		{
			name:   "success",
			handle: defaultHandle,
			nodeID: defaultNodeID,
			caps:   defaultCaps,
			input:  defaultRequest,
			output: &csi.ControllerPublishVolumeResponse{
				Reply: &csi.ControllerPublishVolumeResponse_Result_{
					Result: &csi.ControllerPublishVolumeResponse_Result{
						PublishVolumeInfo: publishVolumeInfo,
					},
				},
			},
			expectError:  false,
			expectedInfo: publishVolumeInfo,
		},
		{
			name:   "success no info",
			handle: defaultHandle,
			nodeID: defaultNodeID,
			caps:   defaultCaps,
			input:  defaultRequest,
			output: &csi.ControllerPublishVolumeResponse{
				Reply: &csi.ControllerPublishVolumeResponse_Result_{
					Result: &csi.ControllerPublishVolumeResponse_Result{},
				},
			},
			expectError:  false,
			expectedInfo: nil,
		},
		{
			name:     "readonly success",
			handle:   defaultHandle,
			nodeID:   defaultNodeID,
			caps:     defaultCaps,
			readonly: true,
			input:    readOnlyRequest,
			output: &csi.ControllerPublishVolumeResponse{
				Reply: &csi.ControllerPublishVolumeResponse_Result_{
					Result: &csi.ControllerPublishVolumeResponse_Result{
						PublishVolumeInfo: publishVolumeInfo,
					},
				},
			},
			expectError:  false,
			expectedInfo: publishVolumeInfo,
		},
		{
			name:        "gRPC error",
			handle:      defaultHandle,
			nodeID:      defaultNodeID,
			caps:        defaultCaps,
			input:       defaultRequest,
			output:      nil,
			injectError: true,
			expectError: true,
		},
		{
			name:   "empty reply",
			handle: defaultHandle,
			nodeID: defaultNodeID,
			caps:   defaultCaps,
			input:  defaultRequest,
			output: &csi.ControllerPublishVolumeResponse{
				Reply: nil,
			},
			expectError: true,
		},
		{
			name:   "general error",
			handle: defaultHandle,
			nodeID: defaultNodeID,
			caps:   defaultCaps,
			input:  defaultRequest,
			output: &csi.ControllerPublishVolumeResponse{
				Reply: &csi.ControllerPublishVolumeResponse_Error{
					Error: &csi.Error{
						Value: &csi.Error_GeneralError_{
							GeneralError: &csi.Error_GeneralError{
								ErrorCode:          csi.Error_GeneralError_UNSUPPORTED_REQUEST_VERSION,
								CallerMustNotRetry: true,
								ErrorDescription:   "mock error 1",
							},
						},
					},
				},
			},
			expectError: true,
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
		if test.injectError {
			injectedErr = fmt.Errorf("mock error")
		}

		// Setup expectation
		if in != nil {
			controllerServer.EXPECT().ControllerPublishVolume(gomock.Any(), in).Return(out, injectedErr).Times(1)
		}

		publishInfo, err := csiConn.Attach(context.Background(), test.handle, test.readonly, test.nodeID, test.caps)
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
		if err == nil && !reflect.DeepEqual(publishInfo, test.expectedInfo) {
			t.Errorf("got unexpected PublishInfo: %+v", publishInfo)
		}
	}
}

func TestDetachAttach(t *testing.T) {
	defaultVolumeName := "myname"
	defaultHandle := &csi.VolumeHandle{
		Id: defaultVolumeName,
	}

	defaultNodeID := &csi.NodeID{Values: map[string]string{"Name": "MyNodeID"}}

	defaultRequest := &csi.ControllerUnpublishVolumeRequest{
		Version: &csiVersion,
		VolumeHandle: &csi.VolumeHandle{
			Id: defaultVolumeName,
		},
		NodeId: defaultNodeID,
	}

	tests := []struct {
		name        string
		handle      *csi.VolumeHandle
		nodeID      *csi.NodeID
		input       *csi.ControllerUnpublishVolumeRequest
		output      *csi.ControllerUnpublishVolumeResponse
		injectError bool
		expectError bool
	}{
		{
			name:   "success",
			handle: defaultHandle,
			nodeID: defaultNodeID,
			input:  defaultRequest,
			output: &csi.ControllerUnpublishVolumeResponse{
				Reply: &csi.ControllerUnpublishVolumeResponse_Result_{
					Result: &csi.ControllerUnpublishVolumeResponse_Result{},
				},
			},
			expectError: false,
		},
		{
			name:        "gRPC error",
			handle:      defaultHandle,
			nodeID:      defaultNodeID,
			input:       defaultRequest,
			output:      nil,
			injectError: true,
			expectError: true,
		},
		{
			name:   "empty reply",
			handle: defaultHandle,
			nodeID: defaultNodeID,
			input:  defaultRequest,
			output: &csi.ControllerUnpublishVolumeResponse{
				Reply: nil,
			},
			expectError: true,
		},
		{
			name:   "general error",
			handle: defaultHandle,
			nodeID: defaultNodeID,
			input:  defaultRequest,
			output: &csi.ControllerUnpublishVolumeResponse{
				Reply: &csi.ControllerUnpublishVolumeResponse_Error{
					Error: &csi.Error{
						Value: &csi.Error_GeneralError_{
							GeneralError: &csi.Error_GeneralError{
								ErrorCode:          csi.Error_GeneralError_UNSUPPORTED_REQUEST_VERSION,
								CallerMustNotRetry: true,
								ErrorDescription:   "mock error 1",
							},
						},
					},
				},
			},
			expectError: true,
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
		if test.injectError {
			injectedErr = fmt.Errorf("mock error")
		}

		// Setup expectation
		if in != nil {
			controllerServer.EXPECT().ControllerUnpublishVolume(gomock.Any(), in).Return(out, injectedErr).Times(1)
		}

		err := csiConn.Detach(context.Background(), test.handle, test.nodeID)
		if test.expectError && err == nil {
			t.Errorf("test %q: Expected error, got none", test.name)
		}
		if !test.expectError && err != nil {
			t.Errorf("test %q: got error: %v", test.name, err)
		}
	}
}
