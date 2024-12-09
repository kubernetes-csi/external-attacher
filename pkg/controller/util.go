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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/container-storage-interface/spec/lib/go/csi"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/kubernetes-csi/csi-lib-utils/accessmodes"
	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

func markAsAttached(ctx context.Context, client kubernetes.Interface, va *storage.VolumeAttachment, metadata map[string]string) (*storage.VolumeAttachment, error) {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Marking as attached")
	clone := va.DeepCopy()
	clone.Status.Attached = true
	clone.Status.AttachmentMetadata = metadata
	clone.Status.AttachError = nil
	patch, err := createMergePatch(va, clone)
	if err != nil {
		return va, err
	}
	newVA, err := client.StorageV1().VolumeAttachments().Patch(ctx, va.Name, types.MergePatchType, patch,
		metav1.PatchOptions{}, "status")
	if err != nil {
		return va, err
	}
	logger.V(4).Info("Marked as attached")
	return newVA, nil
}

func markAsDetached(ctx context.Context, client kubernetes.Interface, va *storage.VolumeAttachment) (*storage.VolumeAttachment, error) {
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

	logger := klog.FromContext(ctx)
	if !found && !va.Status.Attached {
		// Finalizer was not present, nothing to update
		logger.V(4).Info("Already fully detached")
		return va, nil
	}

	logger.V(4).Info("Marking as detached")
	clone := va.DeepCopy()
	clone.Status.Attached = false
	clone.Status.DetachError = nil
	clone.Status.AttachmentMetadata = nil
	patch, err := createMergePatch(va, clone)
	if err != nil {
		return va, err
	}
	newVA, err := client.StorageV1().VolumeAttachments().Patch(ctx, va.Name, types.MergePatchType, patch, metav1.PatchOptions{}, "status")
	if err != nil {
		return va, err
	}

	// As Finalizers is not in the status subresource it must be patched separately. It is removed after the status update so the resource is not prematurely deleted.
	clone = newVA.DeepCopy()
	clone.Finalizers = newFinalizers
	patch, err = createMergePatch(newVA, clone)
	if err != nil {
		return newVA, err
	}
	newVA, err = client.StorageV1().VolumeAttachments().Patch(ctx, newVA.Name, types.MergePatchType, patch, metav1.PatchOptions{}, "")
	if err != nil {
		return newVA, err
	}
	logger.V(4).Info("Finalizer removed")
	return newVA, nil
}

const (
	vaNodeIDAnnotation = "csi.alpha.kubernetes.io/node-id"
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

// GetNodeIDFromCSINode returns nodeID from CSIDriverInfoSpec
func GetNodeIDFromCSINode(driver string, csiNode *storage.CSINode) (string, bool) {
	for _, d := range csiNode.Spec.Drivers {
		if d.Name == driver {
			return d.NodeID, true
		}
	}
	return "", false
}

// GetVolumeCapabilities returns a VolumeCapability from the PV spec. Which access mode will be set depends if the driver supports the
// SINGLE_NODE_MULTI_WRITER capability.
func GetVolumeCapabilities(logger klog.Logger, pvSpec *v1.PersistentVolumeSpec, singleNodeMultiWriterCapable bool, defaultFSType string) (*csi.VolumeCapability, error) {
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
			logger.V(4).Info("Filesystem type not found in PV spec. Using defaultFSType", "defaultFSType", fsType)
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

	am, err := accessmodes.ToCSIAccessMode(pvSpec.AccessModes, singleNodeMultiWriterCapable)
	if err != nil {
		return nil, err
	}

	cap.AccessMode.Mode = am
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
