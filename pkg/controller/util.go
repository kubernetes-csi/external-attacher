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

package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/evanphx/json-patch"
	"k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func markAsAttached(client kubernetes.Interface, va *storage.VolumeAttachment, metadata map[string]string) (*storage.VolumeAttachment, error) {
	klog.V(4).Infof("Marking as attached %q", va.Name)
	clone := va.DeepCopy()
	clone.Status.Attached = true
	clone.Status.AttachmentMetadata = metadata
	clone.Status.AttachError = nil
	patch, err := createMergePatch(va, clone)
	if err != nil {
		return va, err
	}
	newVA, err := client.StorageV1beta1().VolumeAttachments().Patch(va.Name, types.MergePatchType, patch)
	if err != nil {
		return va, err
	}
	klog.V(4).Infof("Marked as attached %q", va.Name)
	return newVA, nil
}

func markAsDetached(client kubernetes.Interface, va *storage.VolumeAttachment) (*storage.VolumeAttachment, error) {
	finalizerName := GetFinalizerName(va.Spec.Attacher)

	// Prepare new array of finalizers
	newFinalizers := make([]string, 0, len(va.Finalizers))
	found := false
	for _, f := range va.Finalizers {
		if f == finalizerName {
			found = true
			continue
		}
		newFinalizers = append(newFinalizers, f)
	}
	// Mostly to simplify unit tests, but it won't harm in production too
	if len(newFinalizers) == 0 {
		newFinalizers = nil
	}

	if !found && !va.Status.Attached {
		// Finalizer was not present, nothing to update
		klog.V(4).Infof("Already fully detached %q", va.Name)
		return va, nil
	}

	klog.V(4).Infof("Marking as detached %q", va.Name)
	clone := va.DeepCopy()
	clone.Finalizers = newFinalizers
	clone.Status.Attached = false
	clone.Status.DetachError = nil
	clone.Status.AttachmentMetadata = nil
	patch, err := createMergePatch(va, clone)
	if err != nil {
		return va, err
	}
	newVA, err := client.StorageV1beta1().VolumeAttachments().Patch(va.Name, types.MergePatchType, patch)
	if err != nil {
		return va, err
	}
	klog.V(4).Infof("Finalizer removed from %q", va.Name)
	return newVA, nil
}

const (
	defaultFSType              = "ext4"
	nodeIDAnnotation           = "csi.volume.kubernetes.io/nodeid"
	csiVolAttribsAnnotationKey = "csi.volume.kubernetes.io/volume-attributes"
	vaNodeIDAnnotation         = "csi.alpha.kubernetes.io/node-id"
)

// SanitizeDriverName sanitizes provided driver name.
func SanitizeDriverName(driver string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9-]")
	name := re.ReplaceAllString(driver, "-")
	if name[len(name)-1] == '-' {
		// name must not end with '-'
		name = name + "X"
	}
	return name
}

// GetFinalizerName returns Attacher name suitable to be used as finalizer
func GetFinalizerName(driver string) string {
	return "external-attacher/" + SanitizeDriverName(driver)
}

// GetNodeIDFromNode returns nodeID string from node annotations.
func GetNodeIDFromNode(driver string, node *v1.Node) (string, error) {
	nodeIDJSON, ok := node.Annotations[nodeIDAnnotation]
	if !ok {
		return "", fmt.Errorf("node %q has no NodeID annotation", node.Name)
	}

	var nodeIDs map[string]string
	if err := json.Unmarshal([]byte(nodeIDJSON), &nodeIDs); err != nil {
		return "", fmt.Errorf("cannot parse NodeID annotation on node %q: %s", node.Name, err)
	}
	nodeID, ok := nodeIDs[driver]
	if !ok {
		return "", fmt.Errorf("cannot find NodeID for driver %q for node %q", driver, node.Name)
	}

	return nodeID, nil
}

// GetNodeIDFromCSINode returns nodeID from CSIDriverInfoSpec
func GetNodeIDFromCSINode(driver string, csiNode *storage.CSINode) (string, bool) {
	for _, d := range csiNode.Spec.Drivers {
		if d.Name == driver {
			return d.NodeID, true
		}
	}
	return "", false
}

// GetVolumeCapabilities returns volumecapability from PV spec
func GetVolumeCapabilities(pvSpec *v1.PersistentVolumeSpec) (*csi.VolumeCapability, error) {
	m := map[v1.PersistentVolumeAccessMode]bool{}
	for _, mode := range pvSpec.AccessModes {
		m[mode] = true
	}

	if pvSpec.CSI == nil {
		return nil, errors.New("CSI volume source was nil")
	}

	var cap *csi.VolumeCapability
	if pvSpec.VolumeMode != nil && *pvSpec.VolumeMode == v1.PersistentVolumeBlock {
		cap = &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Block{
				Block: &csi.VolumeCapability_BlockVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{},
		}

	} else {
		fsType := pvSpec.CSI.FSType
		if len(fsType) == 0 {
			fsType = defaultFSType
		}

		cap = &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{
					FsType:     fsType,
					MountFlags: pvSpec.MountOptions,
				},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{},
		}
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
		return nil, fmt.Errorf("unsupported AccessMode combination: %+v", pvSpec.AccessModes)
	}
	return cap, nil
}

// GetVolumeHandle returns VolumeHandle and Readonly flag from CSI PV source
func GetVolumeHandle(csiSource *v1.CSIPersistentVolumeSource) (string, bool, error) {
	if csiSource == nil {
		return "", false, fmt.Errorf("csi source was nil")
	}
	return csiSource.VolumeHandle, csiSource.ReadOnly, nil
}

// GetVolumeAttributes returns a dictionary of volume attributes from CSI PV source
func GetVolumeAttributes(csiSource *v1.CSIPersistentVolumeSource) (map[string]string, error) {
	if csiSource == nil {
		return nil, fmt.Errorf("csi source was nil")
	}
	return csiSource.VolumeAttributes, nil
}

// createMergePatch return patch generated from original and new interfaces
func createMergePatch(original, new interface{}) ([]byte, error) {
	pvByte, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	cloneByte, err := json.Marshal(new)
	if err != nil {
		return nil, err
	}
	patch, err := jsonpatch.CreateMergePatch(pvByte, cloneByte)
	if err != nil {
		return nil, err
	}
	return patch, nil
}
