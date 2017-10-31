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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestGetNodeID(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expectedID  *csi.NodeID
		expectError bool
	}{
		{
			name:        "single key",
			annotations: map[string]string{"nodeid.csi.volume.kubernetes.io/foo_bar": "MyNodeID"},
			expectedID:  &csi.NodeID{Values: map[string]string{"Name": "MyNodeID"}},
			expectError: false,
		},
		{
			name: "multiple keys",
			annotations: map[string]string{
				"nodeid.csi.volume.kubernetes.io/foo_bar":   "MyNodeID",
				"nodeid.csi.volume.kubernetes.io/foo_bar_":  "MyNodeID1",
				"nodeid.csi.volume.kubernetes.io/_foo_bar_": "MyNodeID2",
			},
			expectedID:  &csi.NodeID{Values: map[string]string{"Name": "MyNodeID"}},
			expectError: false,
		},
		{
			name:        "no annotations",
			annotations: nil,
			expectedID:  nil,
			expectError: true,
		},
		{
			name: "annotations for another driver",
			annotations: map[string]string{
				"nodeid.csi.volume.kubernetes.io/foo_bar_":  "MyNodeID1",
				"nodeid.csi.volume.kubernetes.io/_foo_bar_": "MyNodeID2",
			},
			expectedID:  nil,
			expectError: true,
		},
	}

	for _, test := range tests {
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "abc",
				Annotations: test.annotations,
			},
		}
		nodeID, err := getNodeID(driverName, node)

		if err == nil && test.expectError {
			t.Errorf("test %s: expected error, got none", test.name)
		}
		if err != nil && !test.expectError {
			t.Errorf("test %s: got error: %s", test.name, err)
		}
		if !test.expectError && !reflect.DeepEqual(nodeID, test.expectedID) {
			t.Errorf("test %s: unexpected NodeID: %+v", test.name, nodeID)
		}
	}
}

func createMountCapability(mode csi.VolumeCapability_AccessMode_Mode, mountOptions []string) *csi.VolumeCapability {
	return &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				MountFlags: mountOptions,
			},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: mode,
		},
	}
}
func TestGetVolumeCapabilities(t *testing.T) {
	tests := []struct {
		name               string
		modes              []v1.PersistentVolumeAccessMode
		mountOptions       []string
		expectedCapability *csi.VolumeCapability
		expectError        bool
	}{
		{
			name:               "RWX",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			expectedCapability: createMountCapability(csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER, nil),
			expectError:        false,
		},
		{
			name:               "RWO",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			expectedCapability: createMountCapability(csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER, nil),
			expectError:        false,
		},
		{
			name:               "ROX",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany},
			expectedCapability: createMountCapability(csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY, nil),
			expectError:        false,
		},
		{
			name:               "RWX + anytyhing",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteMany, v1.ReadOnlyMany, v1.ReadWriteOnce},
			expectedCapability: createMountCapability(csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER, nil),
			expectError:        false,
		},
		{
			name:               "mount options",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			expectedCapability: createMountCapability(csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER, []string{"first", "second"}),
			mountOptions:       []string{"first", "second"},
			expectError:        false,
		},
		{
			name:               "ROX+RWO",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany},
			expectedCapability: nil,
			expectError:        true, // not possible in CSI
		},
		{
			name:               "nothing",
			modes:              []v1.PersistentVolumeAccessMode{},
			expectedCapability: nil,
			expectError:        true,
		},
	}

	for _, test := range tests {
		pv := &v1.PersistentVolume{
			Spec: v1.PersistentVolumeSpec{
				AccessModes:  test.modes,
				MountOptions: test.mountOptions,
			},
		}
		cap, err := getVolumeCapabilities(pv)

		if err == nil && test.expectError {
			t.Errorf("test %s: expected error, got none", test.name)
		}
		if err != nil && !test.expectError {
			t.Errorf("test %s: got error: %s", test.name, err)
		}
		if !test.expectError && !reflect.DeepEqual(cap, test.expectedCapability) {
			t.Errorf("test %s: unexpected VolumeCapability: %+v", test.name, cap)
		}
	}
}

func TestAttach(t *testing.T) {
	const defaultVolumeName = "MyVolume1"
	defaultPV := &v1.PersistentVolume{
		Spec: v1.PersistentVolumeSpec{
			AccessModes:  []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			MountOptions: []string{"mount", "options"},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       driverName,
					VolumeHandle: defaultVolumeName,
					ReadOnly:     false,
				},
			},
		},
	}

	nfsPV := &v1.PersistentVolume{
		Spec: v1.PersistentVolumeSpec{
			AccessModes:  []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			MountOptions: []string{"mount", "options"},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{},
			},
		},
	}
	invalidPV := &v1.PersistentVolume{
		Spec: v1.PersistentVolumeSpec{
			AccessModes:  nil, /* no access mode */
			MountOptions: []string{"mount", "options"},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       driverName,
					VolumeHandle: defaultVolumeName,
					ReadOnly:     false,
				},
			},
		},
	}

	defaultNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "abc",
			Annotations: map[string]string{"nodeid.csi.volume.kubernetes.io/foo_bar": "MyNodeID"},
		},
	}
	invalidNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "abc",
			// No NodeID
			Annotations: map[string]string{},
		},
	}

	defaultNodeID := &csi.NodeID{Values: map[string]string{"Name": "MyNodeID"}}
	defaultCaps, _ := getVolumeCapabilities(defaultPV)
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

	tests := []struct {
		name         string
		pv           *v1.PersistentVolume
		node         *v1.Node
		input        *csi.ControllerPublishVolumeRequest
		output       *csi.ControllerPublishVolumeResponse
		injectError  bool
		expectError  bool
		expectedInfo map[string]string
	}{
		{
			name:  "success",
			pv:    defaultPV,
			node:  defaultNode,
			input: defaultRequest,
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
			name:  "success no info",
			pv:    defaultPV,
			node:  defaultNode,
			input: defaultRequest,
			output: &csi.ControllerPublishVolumeResponse{
				Reply: &csi.ControllerPublishVolumeResponse_Result_{
					Result: &csi.ControllerPublishVolumeResponse_Result{},
				},
			},
			expectError:  false,
			expectedInfo: nil,
		},
		{
			name:        "invalid node",
			pv:          defaultPV,
			node:        invalidNode,
			input:       nil,
			output:      nil,
			injectError: false,
			expectError: true,
		},
		{
			name:        "NFS PV",
			pv:          nfsPV,
			node:        defaultNode,
			input:       nil,
			output:      nil,
			injectError: false,
			expectError: true,
		},
		{
			name:        "invalid PV",
			pv:          invalidPV,
			node:        defaultNode,
			input:       nil,
			output:      nil,
			injectError: false,
			expectError: true,
		},
		{
			name:        "gRPC error",
			pv:          defaultPV,
			node:        defaultNode,
			input:       defaultRequest,
			output:      nil,
			injectError: true,
			expectError: true,
		},
		{
			name:  "empty reply",
			pv:    defaultPV,
			node:  defaultNode,
			input: defaultRequest,
			output: &csi.ControllerPublishVolumeResponse{
				Reply: nil,
			},
			expectError: true,
		},
		{
			name:  "general error",
			pv:    defaultPV,
			node:  defaultNode,
			input: defaultRequest,
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

		publishInfo, err := csiConn.Attach(context.Background(), test.pv, test.node)
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
