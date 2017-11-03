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

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corelister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/workqueue"

	"github.com/kubernetes-csi/external-attacher-csi/pkg/connection"
)

// csiHandler is a handler that calls CSI to attach/detach volume.
// It adds finalizer to VolumeAttachment instance to make sure they're detached
// before deletion.
type csiHandler struct {
	client        kubernetes.Interface
	attacherName  string
	csiConnection connection.CSIConnection
	pvLister      corelister.PersistentVolumeLister
	nodeLister    corelister.NodeLister
}

var _ Handler = &csiHandler{}

func NewCSIHandler(
	client kubernetes.Interface,
	attacherName string,
	csiConnection connection.CSIConnection,
	pvLister corelister.PersistentVolumeLister,
	nodeLister corelister.NodeLister) Handler {

	return &csiHandler{
		client,
		attacherName,
		csiConnection,
		pvLister,
		nodeLister,
	}
}

func (h *csiHandler) SyncNewOrUpdatedVolumeAttachment(va *storagev1.VolumeAttachment, queue workqueue.RateLimitingInterface) {
	glog.V(4).Infof("CSIHandler: processing %q", va.Name)

	var err error
	if va.DeletionTimestamp == nil {
		err = h.syncAttach(va)
	} else {
		err = h.syncDetach(va)
	}
	if err != nil {
		// Re-queue with exponential backoff
		glog.V(2).Infof("Error processing %q: %s", va.Name, err)
		queue.AddRateLimited(va.Name)
		return
	}
	// The operation has finished successfully, reset exponential backoff
	queue.Forget(va.Name)
	glog.V(4).Infof("CSIHandler: finished processing %q", va.Name)
}

func (h *csiHandler) syncAttach(va *storagev1.VolumeAttachment) error {
	glog.V(4).Infof("Starting attach operation for %q", va.Name)
	va, err := h.addVAFinalizer(va)
	if err != nil {
		return fmt.Errorf("could not add finalizer: %s", err)
	}

	if va.Status.Attached {
		// Volume is attached, there is nothing to be done.
		glog.V(4).Infof("%q is already attached", va.Name)
		return nil
	}

	// Attach
	glog.V(2).Infof("Attaching %q", va.Name)
	metadata, err := h.csiAttach(va)
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

	glog.V(2).Infof("Detaching %q", va.Name)
	if err := h.csiDetach(va); err != nil {
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
	glog.V(2).Infof("Detached %q", va.Name)

	if _, err := markAsDetached(h.client, va); err != nil {
		return fmt.Errorf("could not mark as detached: %s", err)
	}
	glog.V(4).Infof("Fully detached %q", va.Name)
	return nil
}

func (h *csiHandler) addVAFinalizer(va *storagev1.VolumeAttachment) (*storagev1.VolumeAttachment, error) {
	finalizerName := getFinalizerName(h.attacherName)
	for _, f := range va.Finalizers {
		if f == finalizerName {
			// Finalizer is already present
			glog.V(4).Infof("Finalizer is already set on %q", va.Name)
			return va, nil
		}
	}

	// Finalizer is not present, add it
	glog.V(4).Infof("Adding finalizer to %q", va.Name)
	clone := va.DeepCopy()
	clone.Finalizers = append(clone.Finalizers, finalizerName)
	// TODO: use patch to save us from VersionError
	newVA, err := h.client.StorageV1().VolumeAttachments().Update(clone)
	if err != nil {
		return va, err
	}
	glog.V(4).Infof("Finalizer added to %q", va.Name)
	return newVA, nil
}

func (h *csiHandler) hasVAFinalizer(va *storagev1.VolumeAttachment) bool {
	finalizerName := getFinalizerName(h.attacherName)
	for _, f := range va.Finalizers {
		if f == finalizerName {
			return true
		}
	}
	return false
}

func (h *csiHandler) csiAttach(va *storagev1.VolumeAttachment) (map[string]string, error) {
	if va.Spec.PersistentVolumeName == nil {
		return nil, fmt.Errorf("VolumeAttachment.spec.persistentVolumeName is empty")
	}

	pv, err := h.pvLister.Get(*va.Spec.PersistentVolumeName)
	if err != nil {
		return nil, err
	}
	node, err := h.nodeLister.Get(va.Spec.NodeName)
	if err != nil {
		return nil, err
	}

	ctx := context.TODO()
	publishInfo, err := h.csiConnection.Attach(ctx, pv, node)
	if err != nil {
		return nil, err
	}

	return publishInfo, nil
}

func (h *csiHandler) csiDetach(va *storagev1.VolumeAttachment) error {
	if va.Spec.PersistentVolumeName == nil {
		return fmt.Errorf("VolumeAttachment.spec.persistentVolumeName is empty")
	}

	pv, err := h.pvLister.Get(*va.Spec.PersistentVolumeName)
	if err != nil {
		return err
	}
	node, err := h.nodeLister.Get(va.Spec.NodeName)
	if err != nil {
		return err
	}

	ctx := context.TODO()
	if err := h.csiConnection.Detach(ctx, pv, node); err != nil {
		return err
	}

	return nil
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
