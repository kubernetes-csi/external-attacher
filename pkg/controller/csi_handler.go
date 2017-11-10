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
	"fmt"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corelister "k8s.io/client-go/listers/core/v1"
	storagelister "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/util/workqueue"

	"github.com/kubernetes-csi/external-attacher/pkg/connection"
)

// csiHandler is a handler that calls CSI to attach/detach volume.
// It adds finalizer to VolumeAttachment instance to make sure they're detached
// before deletion.
type csiHandler struct {
	client           kubernetes.Interface
	attacherName     string
	csiConnection    connection.CSIConnection
	pvLister         corelister.PersistentVolumeLister
	nodeLister       corelister.NodeLister
	vaLister         storagelister.VolumeAttachmentLister
	vaQueue, pvQueue workqueue.RateLimitingInterface
}

var _ Handler = &csiHandler{}

func NewCSIHandler(
	client kubernetes.Interface,
	attacherName string,
	csiConnection connection.CSIConnection,
	pvLister corelister.PersistentVolumeLister,
	nodeLister corelister.NodeLister,
	vaLister storagelister.VolumeAttachmentLister) Handler {

	return &csiHandler{
		client:        client,
		attacherName:  attacherName,
		csiConnection: csiConnection,
		pvLister:      pvLister,
		nodeLister:    nodeLister,
		vaLister:      vaLister,
	}
}

func (h *csiHandler) Init(vaQueue workqueue.RateLimitingInterface, pvQueue workqueue.RateLimitingInterface) {
	h.vaQueue = vaQueue
	h.pvQueue = pvQueue
}

func (h *csiHandler) SyncNewOrUpdatedVolumeAttachment(va *storagev1.VolumeAttachment) {
	glog.V(4).Infof("CSIHandler: processing VA %q", va.Name)

	var err error
	if va.DeletionTimestamp == nil {
		err = h.syncAttach(va)
	} else {
		err = h.syncDetach(va)
	}
	if err != nil {
		// Re-queue with exponential backoff
		glog.V(2).Infof("Error processing %q: %s", va.Name, err)
		h.vaQueue.AddRateLimited(va.Name)
		return
	}
	// The operation has finished successfully, reset exponential backoff
	h.vaQueue.Forget(va.Name)
	glog.V(4).Infof("CSIHandler: finished processing %q", va.Name)
}

func (h *csiHandler) syncAttach(va *storagev1.VolumeAttachment) error {
	if va.Status.Attached {
		// Volume is attached, there is nothing to be done.
		glog.V(4).Infof("%q is already attached", va.Name)
		return nil
	}

	// Attach and report any error
	glog.V(2).Infof("Attaching %q", va.Name)
	va, metadata, err := h.csiAttach(va)
	if err != nil {
		var saveErr error
		va, saveErr = h.saveAttachError(va, err)
		if saveErr != nil {
			// Just log it, propagate the attach error.
			glog.V(2).Infof("Failed to save attach error to %q: %s", va.Name, saveErr.Error())
		}
		// Add context to the error for logging
		err := fmt.Errorf("failed to attach: %s", err)
		return err
	}
	glog.V(2).Infof("Attached %q", va.Name)

	// Mark as attached
	if _, err := markAsAttached(h.client, va, metadata); err != nil {
		return fmt.Errorf("failed to mark as attached: %s", err)
	}
	glog.V(4).Infof("Fully attached %q", va.Name)
	return nil
}

func (h *csiHandler) syncDetach(va *storagev1.VolumeAttachment) error {
	glog.V(4).Infof("Starting detach operation for %q", va.Name)
	if !h.hasVAFinalizer(va) {
		glog.V(4).Infof("%q is already detached", va.Name)
		return nil
	}

	// Detach and report any error
	glog.V(2).Infof("Detaching %q", va.Name)
	va, err := h.csiDetach(va)
	if err != nil {
		var saveErr error
		va, saveErr = h.saveDetachError(va, err)
		if saveErr != nil {
			// Just log it, propagate the detach error.
			glog.V(2).Infof("Failed to save detach error to %q: %s", va.Name, saveErr.Error())
		}
		// Add context to the error for logging
		err := fmt.Errorf("failed to detach: %s", err)
		return err
	}
	glog.V(4).Infof("Fully detached %q", va.Name)
	return nil
}

func (h *csiHandler) addVAFinalizer(va *storagev1.VolumeAttachment) (*storagev1.VolumeAttachment, error) {
	finalizerName := connection.GetFinalizerName(h.attacherName)
	for _, f := range va.Finalizers {
		if f == finalizerName {
			// Finalizer is already present
			glog.V(4).Infof("VA finalizer is already set on %q", va.Name)
			return va, nil
		}
	}

	// Finalizer is not present, add it
	glog.V(4).Infof("Adding finalizer to VA %q", va.Name)
	clone := va.DeepCopy()
	clone.Finalizers = append(clone.Finalizers, finalizerName)
	// TODO: use patch to save us from VersionError
	newVA, err := h.client.StorageV1().VolumeAttachments().Update(clone)
	if err != nil {
		return va, err
	}
	glog.V(4).Infof("VA finalizer added to %q", va.Name)
	return newVA, nil
}

func (h *csiHandler) addPVFinalizer(pv *v1.PersistentVolume) (*v1.PersistentVolume, error) {
	finalizerName := connection.GetFinalizerName(h.attacherName)
	for _, f := range pv.Finalizers {
		if f == finalizerName {
			// Finalizer is already present
			glog.V(4).Infof("PV finalizer is already set on %q", pv.Name)
			return pv, nil
		}
	}

	// Finalizer is not present, add it
	glog.V(4).Infof("Adding finalizer to PV %q", pv.Name)
	clone := pv.DeepCopy()
	clone.Finalizers = append(clone.Finalizers, finalizerName)
	// TODO: use patch to save us from VersionError
	newPV, err := h.client.CoreV1().PersistentVolumes().Update(clone)
	if err != nil {
		return pv, err
	}
	glog.V(4).Infof("PV finalizer added to %q", pv.Name)
	return newPV, nil
}

func (h *csiHandler) hasVAFinalizer(va *storagev1.VolumeAttachment) bool {
	finalizerName := connection.GetFinalizerName(h.attacherName)
	for _, f := range va.Finalizers {
		if f == finalizerName {
			return true
		}
	}
	return false
}

func (h *csiHandler) csiAttach(va *storagev1.VolumeAttachment) (*storagev1.VolumeAttachment, map[string]string, error) {
	glog.V(4).Infof("Starting attach operation for %q", va.Name)
	// Check as much as possible before adding VA finalizer - it would block
	// deletion of VA on error.

	if va.Spec.PersistentVolumeName == nil {
		return va, nil, fmt.Errorf("VolumeAttachment.spec.persistentVolumeName is empty")
	}

	pv, err := h.pvLister.Get(*va.Spec.PersistentVolumeName)
	if err != nil {
		return va, nil, err
	}
	// Refuse to attach volumes that are marked for deletion.
	if pv.DeletionTimestamp != nil {
		return va, nil, fmt.Errorf("PersistentVolume %q is marked for deletion", pv.Name)
	}
	pv, err = h.addPVFinalizer(pv)
	if err != nil {
		return va, nil, fmt.Errorf("could not add PersistentVolume finalizer: %s", err)
	}

	volumeHandle, readOnly, err := connection.GetVolumeHandle(pv)
	if err != nil {
		return va, nil, err
	}
	volumeCapabilities, err := connection.GetVolumeCapabilities(pv)
	if err != nil {
		return va, nil, err
	}

	node, err := h.nodeLister.Get(va.Spec.NodeName)
	if err != nil {
		return va, nil, err
	}
	nodeID, err := connection.GetNodeID(h.attacherName, node)
	if err != nil {
		return va, nil, err
	}

	va, err = h.addVAFinalizer(va)
	if err != nil {
		return va, nil, fmt.Errorf("could not add VolumeAttachment finalizer: %s", err)
	}

	ctx := context.TODO()
	publishInfo, err := h.csiConnection.Attach(ctx, volumeHandle, readOnly, nodeID, volumeCapabilities)
	if err != nil {
		return va, nil, err
	}

	return va, publishInfo, nil
}

func (h *csiHandler) csiDetach(va *storagev1.VolumeAttachment) (*storagev1.VolumeAttachment, error) {
	if va.Spec.PersistentVolumeName == nil {
		return va, fmt.Errorf("VolumeAttachment.spec.persistentVolumeName is empty")
	}

	pv, err := h.pvLister.Get(*va.Spec.PersistentVolumeName)
	if err != nil {
		return va, err
	}
	volumeHandle, _, err := connection.GetVolumeHandle(pv)
	if err != nil {
		return va, err
	}

	node, err := h.nodeLister.Get(va.Spec.NodeName)
	if err != nil {
		return va, err
	}
	nodeID, err := connection.GetNodeID(h.attacherName, node)
	if err != nil {
		return va, err
	}

	ctx := context.TODO()
	if err := h.csiConnection.Detach(ctx, volumeHandle, nodeID); err != nil {
		return va, err
	}
	glog.V(2).Infof("Detached %q", va.Name)

	if va, err := markAsDetached(h.client, va); err != nil {
		return va, fmt.Errorf("could not mark as detached: %s", err)
	}

	return va, nil
}

func (h *csiHandler) saveAttachError(va *storagev1.VolumeAttachment, err error) (*storagev1.VolumeAttachment, error) {
	glog.V(4).Infof("Saving attach error to %q", va.Name)
	clone := va.DeepCopy()
	clone.Status.AttachError = &storagev1.VolumeError{
		Message: err.Error(),
		Time:    metav1.Now(),
	}
	newVa, err := h.client.StorageV1().VolumeAttachments().Update(clone)
	if err != nil {
		return va, err
	}
	glog.V(4).Infof("Saved attach error to %q", va.Name)
	return newVa, nil
}

func (h *csiHandler) saveDetachError(va *storagev1.VolumeAttachment, err error) (*storagev1.VolumeAttachment, error) {
	glog.V(4).Infof("Saving detach error to %q", va.Name)
	clone := va.DeepCopy()
	clone.Status.DetachError = &storagev1.VolumeError{
		Message: err.Error(),
		Time:    metav1.Now(),
	}
	newVa, err := h.client.StorageV1().VolumeAttachments().Update(clone)
	if err != nil {
		return va, err
	}
	glog.V(4).Infof("Saved detach error to %q", va.Name)
	return newVa, nil
}

func (h *csiHandler) SyncNewOrUpdatedPersistentVolume(pv *v1.PersistentVolume) {
	glog.V(4).Infof("CSIHandler: processing PV %q", pv.Name)
	// Sync and remove finalizer on given PV
	if pv.DeletionTimestamp == nil {
		// Don't process anything that has no deletion timestamp.
		glog.V(4).Infof("CSIHandler: processing PV %q: no deletion timestamp, ignoring", pv.Name)
		h.pvQueue.Forget(pv.Name)
		return
	}

	// Check if the PV has finalizer
	finalizer := connection.GetFinalizerName(h.attacherName)
	found := false
	for _, f := range pv.Finalizers {
		if f == finalizer {
			found = true
			break
		}
	}
	if !found {
		// No finalizer -> no action required
		glog.V(4).Infof("CSIHandler: processing PV %q: no finalizer, ignoring", pv.Name)
		h.pvQueue.Forget(pv.Name)
		return
	}

	// Check that there is no VA that requires the PV
	vas, err := h.vaLister.List(labels.Everything())
	if err != nil {
		// Failed listing VAs? Try again with exp. backoff
		glog.Errorf("Failed to list VolumeAttachments for PV %q: %s", pv.Name, err.Error())
		h.pvQueue.AddRateLimited(pv.Name)
		return
	}
	for _, va := range vas {
		if va.Spec.AttachedVolumeSource.PersistentVolumeName != nil && *va.Spec.AttachedVolumeSource.PersistentVolumeName == pv.Name {
			// This PV is needed by this VA, don't remove finalizer
			glog.V(4).Infof("CSIHandler: processing PV %q: VA %q found", pv.Name, va.Name)
			h.pvQueue.Forget(pv.Name)
			return
		}
	}
	// No VA found -> remove finalizer
	glog.V(4).Infof("CSIHandler: processing PV %q: no VA found, removing finalizer", pv.Name)
	clone := pv.DeepCopy()
	newFinalizers := []string{}
	for _, f := range pv.Finalizers {
		if f == finalizer {
			continue
		}
		newFinalizers = append(newFinalizers, f)
	}
	if len(newFinalizers) == 0 {
		// Canonize empty finalizers for unit test (so we don't need to
		// distinguish nil and [] there)
		newFinalizers = nil
	}
	clone.Finalizers = newFinalizers
	_, err = h.client.CoreV1().PersistentVolumes().Update(clone)
	if err != nil {
		glog.Errorf("Failed to remove finalizer from PV %q: %s", pv.Name, err.Error())
		h.pvQueue.AddRateLimited(pv.Name)
		return
	}
	glog.V(2).Infof("Removed finalizer from PV %q", pv.Name)
	h.pvQueue.Forget(pv.Name)

	return
}
