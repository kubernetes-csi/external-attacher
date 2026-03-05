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

package controller

import (
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2/ktesting"
	_ "k8s.io/klog/v2/ktesting/init"
)

func createBlockCapability(mode csi.VolumeCapability_AccessMode_Mode) *csi.VolumeCapability {
	return &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Block{
			Block: &csi.VolumeCapability_BlockVolume{},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: mode,
		},
	}
}

func createMountCapability(fsType string, mode csi.VolumeCapability_AccessMode_Mode, mountOptions []string) *csi.VolumeCapability {
	return &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				FsType:     fsType,
				MountFlags: mountOptions,
			},
		},
		AccessMode: &csi.VolumeCapability_AccessMode{
			Mode: mode,
		},
	}
}

func TestGetVolumeCapabilities(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	blockVolumeMode := v1.PersistentVolumeMode(v1.PersistentVolumeBlock)
	filesystemVolumeMode := v1.PersistentVolumeMode(v1.PersistentVolumeFilesystem)
	defaultFSType := "ext4"

	tests := []struct {
		name                          string
		volumeMode                    *v1.PersistentVolumeMode
		fsType                        string
		modes                         []v1.PersistentVolumeAccessMode
		mountOptions                  []string
		supportsSingleNodeMultiWriter bool
		expectedCapability            *csi.VolumeCapability
		expectError                   bool
	}{
		{
			name:               "RWX",
			volumeMode:         &filesystemVolumeMode,
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			expectedCapability: createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER, nil),
			expectError:        false,
		},
		{
			name:               "Block RWX",
			volumeMode:         &blockVolumeMode,
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			expectedCapability: createBlockCapability(csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER),
			expectError:        false,
		},
		{
			name:               "RWX + specified fsType",
			fsType:             "ext3",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			expectedCapability: createMountCapability("ext3", csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER, nil),
			expectError:        false,
		},
		{
			name:               "RWO",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			expectedCapability: createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER, nil),
			expectError:        false,
		},
		{
			name:               "Block RWO",
			volumeMode:         &blockVolumeMode,
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			expectedCapability: createBlockCapability(csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER),
			expectError:        false,
		},
		{
			name:               "ROX",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany},
			expectedCapability: createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY, nil),
			expectError:        false,
		},
		{
			name:               "RWX + anything",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteMany, v1.ReadOnlyMany, v1.ReadWriteOnce},
			expectedCapability: createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER, nil),
			expectError:        false,
		},
		{
			name:               "mount options",
			modes:              []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			expectedCapability: createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER, []string{"first", "second"}),
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
		{
			name:                          "RWX with SINGLE_NODE_MULTI_WRITER capable driver",
			volumeMode:                    &filesystemVolumeMode,
			modes:                         []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			supportsSingleNodeMultiWriter: true,
			expectedCapability:            createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER, nil),
			expectError:                   false,
		},
		{
			name:                          "ROX + RWO with SINGLE_NODE_MULTI_WRITER capable driver",
			volumeMode:                    &filesystemVolumeMode,
			modes:                         []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany, v1.ReadWriteOnce},
			supportsSingleNodeMultiWriter: true,
			expectedCapability:            nil,
			expectError:                   true,
		},
		{
			name:                          "ROX + RWOP with SINGLE_NODE_MULTI_WRITER capable driver",
			volumeMode:                    &filesystemVolumeMode,
			modes:                         []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany, v1.ReadWriteOncePod},
			supportsSingleNodeMultiWriter: true,
			expectedCapability:            nil,
			expectError:                   true,
		},
		{
			name:                          "RWO + RWOP with SINGLE_NODE_MULTI_WRITER capable driver",
			volumeMode:                    &filesystemVolumeMode,
			modes:                         []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadWriteOncePod},
			supportsSingleNodeMultiWriter: true,
			expectedCapability:            nil,
			expectError:                   true,
		},
		{
			name:                          "ROX with SINGLE_NODE_MULTI_WRITER capable driver",
			volumeMode:                    &filesystemVolumeMode,
			modes:                         []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany},
			supportsSingleNodeMultiWriter: true,
			expectedCapability:            createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY, nil),
			expectError:                   false,
		},
		{
			name:                          "RWO with SINGLE_NODE_MULTI_WRITER capable driver",
			volumeMode:                    &filesystemVolumeMode,
			modes:                         []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			supportsSingleNodeMultiWriter: true,
			expectedCapability:            createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_SINGLE_NODE_MULTI_WRITER, nil),
			expectError:                   false,
		},
		{
			name:                          "RWOP with SINGLE_NODE_MULTI_WRITER capable driver",
			volumeMode:                    &filesystemVolumeMode,
			modes:                         []v1.PersistentVolumeAccessMode{v1.ReadWriteOncePod},
			supportsSingleNodeMultiWriter: true,
			expectedCapability:            createMountCapability(defaultFSType, csi.VolumeCapability_AccessMode_SINGLE_NODE_SINGLE_WRITER, nil),
			expectError:                   false,
		},
		{
			name:                          "nothing with SINGLE_NODE_MULTI_WRITER capable driver",
			modes:                         []v1.PersistentVolumeAccessMode{},
			supportsSingleNodeMultiWriter: true,
			expectedCapability:            nil,
			expectError:                   true,
		},
	}

	for _, test := range tests {
		pv := &v1.PersistentVolume{
			Spec: v1.PersistentVolumeSpec{
				VolumeMode:   test.volumeMode,
				AccessModes:  test.modes,
				MountOptions: test.mountOptions,
				PersistentVolumeSource: v1.PersistentVolumeSource{
					CSI: &v1.CSIPersistentVolumeSource{
						FSType: test.fsType,
					},
				},
			},
		}
		cap, err := GetVolumeCapabilities(logger, &pv.Spec, test.supportsSingleNodeMultiWriter, defaultFSType)

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

func TestSanitizeDriverName(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{
			"no-Change",
			"no-Change",
		},
		{
			"not!allowed/characters",
			"not-allowed-characters",
		},
		{
			"trailing\\",
			"trailing-X",
		},
	}

	for _, test := range tests {
		output := SanitizeDriverName(test.input)
		if output != test.output {
			t.Errorf("expected %q, got %q", test.output, output)
		}
	}
}

func TestGetFinalizerName(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{
			"no-Change",
			"external-attacher/no-Change",
		},
		{
			"s!a@n#i$t(i%z^e&d*",
			"external-attacher/s-a-n-i-t-i-z-e-d-X",
		},
	}

	for _, test := range tests {
		output := GetFinalizerName(test.input)
		if output != test.output {
			t.Errorf("expected %q, got %q", test.output, output)
		}
	}
}

func TestAddFinalizerPatch(t *testing.T) {
	tests := []struct {
		name       string
		finalizers []string
		finalizer  string
		wantPatch  string
		wantErr    bool
	}{
		{
			name:       "add to nil finalizers",
			finalizers: nil,
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantPatch:  `[{"op":"test","path":"/metadata/finalizers","value":null},{"op":"add","path":"/metadata/finalizers","value":["external-attacher/ebs-csi-aws-com"]}]`,
		},
		{
			name:       "add to empty finalizers",
			finalizers: []string{},
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantPatch:  `[{"op":"test","path":"/metadata/finalizers","value":[]},{"op":"replace","path":"/metadata/finalizers","value":["external-attacher/ebs-csi-aws-com"]}]`,
		},
		{
			name:       "add to existing finalizers preserves others",
			finalizers: []string{"external-provisioner.volume.kubernetes.io/finalizer", "kubernetes.io/pv-protection"},
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantPatch:  `[{"op":"add","path":"/metadata/finalizers/-","value":"external-attacher/ebs-csi-aws-com"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addFinalizerPatch(tt.finalizers, tt.finalizer)
			if (err != nil) != tt.wantErr {
				t.Fatalf("addFinalizerPatch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if string(got) != tt.wantPatch {
				t.Errorf("addFinalizerPatch() =\n  %s\nwant:\n  %s", string(got), tt.wantPatch)
			}
		})
	}
}

func TestRemoveFinalizerPatch(t *testing.T) {
	tests := []struct {
		name       string
		finalizers []string
		finalizer  string
		wantPatch  string
		wantErr    bool
	}{
		{
			name:       "remove sole finalizer",
			finalizers: []string{"external-attacher/ebs-csi-aws-com"},
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantPatch:  `[{"op":"test","path":"/metadata/finalizers/0","value":"external-attacher/ebs-csi-aws-com"},{"op":"remove","path":"/metadata/finalizers/0"}]`,
		},
		{
			name:       "remove first of many",
			finalizers: []string{"external-attacher/ebs-csi-aws-com", "kubernetes.io/pv-protection", "external-provisioner.volume.kubernetes.io/finalizer"},
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantPatch:  `[{"op":"test","path":"/metadata/finalizers/0","value":"external-attacher/ebs-csi-aws-com"},{"op":"remove","path":"/metadata/finalizers/0"}]`,
		},
		{
			name:       "remove middle finalizer leaves others untouched",
			finalizers: []string{"external-provisioner.volume.kubernetes.io/finalizer", "external-attacher/ebs-csi-aws-com", "kubernetes.io/pv-protection"},
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantPatch:  `[{"op":"test","path":"/metadata/finalizers/1","value":"external-attacher/ebs-csi-aws-com"},{"op":"remove","path":"/metadata/finalizers/1"}]`,
		},
		{
			name:       "remove last finalizer",
			finalizers: []string{"kubernetes.io/pv-protection", "external-attacher/ebs-csi-aws-com"},
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantPatch:  `[{"op":"test","path":"/metadata/finalizers/1","value":"external-attacher/ebs-csi-aws-com"},{"op":"remove","path":"/metadata/finalizers/1"}]`,
		},
		{
			name:       "finalizer not found",
			finalizers: []string{"kubernetes.io/pv-protection"},
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantErr:    true,
		},
		{
			name:       "nil finalizers",
			finalizers: nil,
			finalizer:  "external-attacher/ebs-csi-aws-com",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := removeFinalizerPatch(tt.finalizers, tt.finalizer)
			if (err != nil) != tt.wantErr {
				t.Fatalf("removeFinalizerPatch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && string(got) != tt.wantPatch {
				t.Errorf("removeFinalizerPatch() =\n  %s\nwant:\n  %s", string(got), tt.wantPatch)
			}
		})
	}
}

func TestGetVolumeHandle(t *testing.T) {
	pv := &v1.PersistentVolume{
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       "myDriver",
					VolumeHandle: "name",
					ReadOnly:     false,
				},
			},
		},
	}

	validPV := pv.DeepCopy()

	readOnlyPV := pv.DeepCopy()
	readOnlyPV.Spec.PersistentVolumeSource.CSI.ReadOnly = true

	invalidPV := pv.DeepCopy()
	invalidPV.Spec.PersistentVolumeSource.CSI = nil

	tests := []struct {
		pv          *v1.PersistentVolume
		output      string
		readOnly    bool
		expectError bool
	}{
		{
			pv:     validPV,
			output: "name",
		},
		{
			pv:       readOnlyPV,
			output:   "name",
			readOnly: true,
		},
		{
			pv:          invalidPV,
			output:      "",
			expectError: true,
		},
	}

	for i, test := range tests {
		output, readOnly, err := GetVolumeHandle(test.pv.Spec.CSI)
		if output != test.output {
			t.Errorf("test %d: expected volume ID %q, got %q", i, test.output, output)
		}
		if readOnly != test.readOnly {
			t.Errorf("test %d: expected readonly %v, got %v", i, test.readOnly, readOnly)
		}
		if err == nil && test.expectError {
			t.Errorf("test %d: expected error, got none", i)
		}
		if err != nil && !test.expectError {
			t.Errorf("test %d: got error %s", i, err)
		}
	}
}
