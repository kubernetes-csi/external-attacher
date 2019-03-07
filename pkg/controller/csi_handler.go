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
	"time"

	"k8s.io/klog"

	"github.com/kubernetes-csi/external-attacher/pkg/attacher"
	"k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1beta1"
	"k8s.io/client-go/util/workqueue"
	csiclient "k8s.io/csi-api/pkg/client/clientset/versioned"
	csilisters "k8s.io/csi-api/pkg/client/listers/csi/v1alpha1"
	csitranslationlib "k8s.io/csi-translation-lib"
)

// csiHandler is a handler that calls CSI to attach/detach volume.
// It adds finalizer to VolumeAttachment instance to make sure they're detached
// before deletion.
type csiHandler struct {
	client                  kubernetes.Interface
	csiClientSet            csiclient.Interface
	attacherName            string
	attacher                attacher.Attacher
	pvLister                corelisters.PersistentVolumeLister
	nodeLister              corelisters.NodeLister
	nodeInfoLister          csilisters.CSINodeInfoLister
	vaLister                storagelisters.VolumeAttachmentLister
	vaQueue, pvQueue        workqueue.RateLimitingInterface
	timeout                 time.Duration
	supportsPublishReadOnly bool
}

var _ Handler = &csiHandler{}

// NewCSIHandler creates a new CSIHandler.
func NewCSIHandler(
	client kubernetes.Interface,
	csiClientSet csiclient.Interface,
	attacherName string,
	attacher attacher.Attacher,
	pvLister corelisters.PersistentVolumeLister,
	nodeLister corelisters.NodeLister,
	nodeInfoLister csilisters.CSINodeInfoLister,
	vaLister storagelisters.VolumeAttachmentLister,
	timeout *time.Duration,
	supportsPublishReadOnly bool) Handler {

	return &csiHandler{
		client:                  client,
		csiClientSet:            csiClientSet,
		attacherName:            attacherName,
		attacher:                attacher,
		pvLister:                pvLister,
		nodeLister:              nodeLister,
		nodeInfoLister:          nodeInfoLister,
		vaLister:                vaLister,
		timeout:                 *timeout,
		supportsPublishReadOnly: supportsPublishReadOnly,
	}
}

func (h *csiHandler) Init(vaQueue workqueue.RateLimitingInterface, pvQueue workqueue.RateLimitingInterface) {
	h.vaQueue = vaQueue
	h.pvQueue = pvQueue
}

func (h *csiHandler) SyncNewOrUpdatedVolumeAttachment(va *storage.VolumeAttachment) {
	klog.V(4).Infof("CSIHandler: processing VA %q", va.Name)

	var err error
	if va.DeletionTimestamp == nil {
		err = h.syncAttach(va)
	} else {
		err = h.syncDetach(va)
	}
	if err != nil {
		// Re-queue with exponential backoff
		klog.V(2).Infof("Error processing %q: %s", va.Name, err)
		h.vaQueue.AddRateLimited(va.Name)
		return
	}
	// The operation has finished successfully, reset exponential backoff
	h.vaQueue.Forget(va.Name)
	klog.V(4).Infof("CSIHandler: finished processing %q", va.Name)
}

func (h *csiHandler) syncAttach(va *storage.VolumeAttachment) error {
	if va.Status.Attached {
		// Volume is attached, there is nothing to be done.
		klog.V(4).Infof("%q is already attached", va.Name)
		return nil
	}

	// Attach and report any error
	klog.V(2).Infof("Attaching %q", va.Name)
	va, metadata, err := h.csiAttach(va)
	if err != nil {
		var saveErr error
		va, saveErr = h.saveAttachError(va, err)
		if saveErr != nil {
			// Just log it, propagate the attach error.
			klog.V(2).Infof("Failed to save attach error to %q: %s", va.Name, saveErr.Error())
		}
		// Add context to the error for logging
		err := fmt.Errorf("failed to attach: %s", err)
		return err
	}
	klog.V(2).Infof("Attached %q", va.Name)

	// Mark as attached
	if _, err := markAsAttached(h.client, va, metadata); err != nil {
		return fmt.Errorf("failed to mark as attached: %s", err)
	}
	klog.V(4).Infof("Fully attached %q", va.Name)
	return nil
}

func (h *csiHandler) syncDetach(va *storage.VolumeAttachment) error {
	klog.V(4).Infof("Starting detach operation for %q", va.Name)
	if !h.hasVAFinalizer(va) {
		klog.V(4).Infof("%q is already detached", va.Name)
		return nil
	}

	// Detach and report any error
	klog.V(2).Infof("Detaching %q", va.Name)
	va, err := h.csiDetach(va)
	if err != nil {
		var saveErr error
		va, saveErr = h.saveDetachError(va, err)
		if saveErr != nil {
			// Just log it, propagate the detach error.
			klog.V(2).Infof("Failed to save detach error to %q: %s", va.Name, saveErr.Error())
		}
		// Add context to the error for logging
		err := fmt.Errorf("failed to detach: %s", err)
		return err
	}
	klog.V(4).Infof("Fully detached %q", va.Name)
	return nil
}

func (h *csiHandler) prepareVAFinalizer(va *storage.VolumeAttachment) (newVA *storage.VolumeAttachment, modified bool) {
	finalizerName := GetFinalizerName(h.attacherName)
	for _, f := range va.Finalizers {
		if f == finalizerName {
			// Finalizer is already present
			klog.V(4).Infof("VA finalizer is already set on %q", va.Name)
			return va, false
		}
	}

	// Finalizer is not present, add it
	clone := va.DeepCopy()
	clone.Finalizers = append(clone.Finalizers, finalizerName)
	klog.V(4).Infof("VA finalizer added to %q", va.Name)
	return clone, true
}

func (h *csiHandler) prepareVANodeID(va *storage.VolumeAttachment, nodeID string) (newVA *storage.VolumeAttachment, modified bool) {
	if existingID, ok := va.Annotations[vaNodeIDAnnotation]; ok && existingID == nodeID {
		klog.V(4).Infof("NodeID annotation is already set on %q", va.Name)
		return va, false
	}
	clone := va.DeepCopy()
	if clone.Annotations == nil {
		clone.Annotations = map[string]string{}
	}
	clone.Annotations[vaNodeIDAnnotation] = nodeID
	klog.V(4).Infof("NodeID annotation added to %q", va.Name)
	return clone, true
}

func (h *csiHandler) saveVA(va *storage.VolumeAttachment) (*storage.VolumeAttachment, error) {
	// TODO: use patch to save us from VersionError
	newVA, err := h.client.StorageV1beta1().VolumeAttachments().Update(va)
	if err != nil {
		return va, err
	}
	klog.V(4).Infof("VolumeAttachment %q updated with finalizer and/or NodeID annotation", va.Name)
	return newVA, nil
}

func (h *csiHandler) addPVFinalizer(pv *v1.PersistentVolume) (*v1.PersistentVolume, error) {
	finalizerName := GetFinalizerName(h.attacherName)
	for _, f := range pv.Finalizers {
		if f == finalizerName {
			// Finalizer is already present
			klog.V(4).Infof("PV finalizer is already set on %q", pv.Name)
			return pv, nil
		}
	}

	// Finalizer is not present, add it
	klog.V(4).Infof("Adding finalizer to PV %q", pv.Name)
	clone := pv.DeepCopy()
	clone.Finalizers = append(clone.Finalizers, finalizerName)
	// TODO: use patch to save us from VersionError
	newPV, err := h.client.CoreV1().PersistentVolumes().Update(clone)
	if err != nil {
		return pv, err
	}
	klog.V(4).Infof("PV finalizer added to %q", pv.Name)
	return newPV, nil
}

func (h *csiHandler) hasVAFinalizer(va *storage.VolumeAttachment) bool {
	finalizerName := GetFinalizerName(h.attacherName)
	for _, f := range va.Finalizers {
		if f == finalizerName {
			return true
		}
	}
	return false
}

func getCSISource(pv *v1.PersistentVolume) (*v1.CSIPersistentVolumeSource, error) {
	if pv == nil {
		return nil, fmt.Errorf("could not get CSI source, pv was nil")
	}
	if pv.Spec.CSI != nil {
		return pv.Spec.CSI, nil
	} else if csitranslationlib.IsPVMigratable(pv) {
		csiPV, err := csitranslationlib.TranslateInTreePVToCSI(pv)
		if err != nil {
			return nil, fmt.Errorf("failed to translate in tree pv to CSI: %v", err)
		}
		return csiPV.Spec.CSI, nil
	}
	return nil, fmt.Errorf("pv contained non-csi source that was not migrated")
}

func (h *csiHandler) csiAttach(va *storage.VolumeAttachment) (*storage.VolumeAttachment, map[string]string, error) {
	klog.V(4).Infof("Starting attach operation for %q", va.Name)
	// Check as much as possible before adding VA finalizer - it would block
	// deletion of VA on error.

	if va.Spec.Source.PersistentVolumeName == nil {
		return va, nil, fmt.Errorf("VolumeAttachment.spec.persistentVolumeName is empty")
	}

	pv, err := h.pvLister.Get(*va.Spec.Source.PersistentVolumeName)
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

	csiSource, err := getCSISource(pv)
	if err != nil {
		return va, nil, err
	}

	attributes, err := GetVolumeAttributes(csiSource)
	if err != nil {
		return va, nil, err
	}

	volumeHandle, readOnly, err := GetVolumeHandle(csiSource)
	if err != nil {
		return va, nil, err
	}
	if !h.supportsPublishReadOnly {
		// "CO MUST set this field to false if SP does not have the
		// PUBLISH_READONLY controller capability"
		readOnly = false
	}

	volumeCapabilities, err := GetVolumeCapabilities(pv, csiSource)
	if err != nil {
		return va, nil, err
	}
	secrets, err := h.getCredentialsFromPV(csiSource)
	if err != nil {
		return va, nil, err
	}

	nodeID, err := h.getNodeID(h.attacherName, va.Spec.NodeName, nil)
	if err != nil {
		return va, nil, err
	}

	originalVA := va
	va, finalizerAdded := h.prepareVAFinalizer(va)
	va, nodeIDAdded := h.prepareVANodeID(va, nodeID)
	if finalizerAdded || nodeIDAdded {
		va, err = h.saveVA(va)
		if err != nil {
			// va modification failed, return the original va that's still on API server
			return originalVA, nil, fmt.Errorf("could not save VolumeAttachment: %s", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	// We're not interested in `detached` return value, the controller will
	// issue Detach to be sure the volume is really detached.
	publishInfo, _, err := h.attacher.Attach(ctx, volumeHandle, readOnly, nodeID, volumeCapabilities, attributes, secrets)
	if err != nil {
		return va, nil, err
	}

	return va, publishInfo, nil
}

func (h *csiHandler) csiDetach(va *storage.VolumeAttachment) (*storage.VolumeAttachment, error) {
	if va.Spec.Source.PersistentVolumeName == nil {
		return va, fmt.Errorf("VolumeAttachment.spec.persistentVolumeName is empty")
	}

	pv, err := h.pvLister.Get(*va.Spec.Source.PersistentVolumeName)
	if err != nil {
		return va, err
	}

	csiSource, err := getCSISource(pv)
	if err != nil {
		return va, err
	}

	volumeHandle, _, err := GetVolumeHandle(csiSource)
	if err != nil {
		return va, err
	}
	secrets, err := h.getCredentialsFromPV(csiSource)
	if err != nil {
		return va, err
	}

	nodeID, err := h.getNodeID(h.attacherName, va.Spec.NodeName, va)
	if err != nil {
		return va, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	detached, err := h.attacher.Detach(ctx, volumeHandle, nodeID, secrets)
	if err != nil && !detached {
		// The volume may not be fully detached. Save the error and try again
		// after backoff.
		return va, err
	}
	if err != nil {
		klog.V(2).Infof("Detached %q with error %s", va.Name, err.Error())
	} else {
		klog.V(2).Infof("Detached %q", va.Name)
	}

	if va, err := markAsDetached(h.client, va); err != nil {
		return va, fmt.Errorf("could not mark as detached: %s", err)
	}

	return va, nil
}

func (h *csiHandler) saveAttachError(va *storage.VolumeAttachment, err error) (*storage.VolumeAttachment, error) {
	klog.V(4).Infof("Saving attach error to %q", va.Name)
	clone := va.DeepCopy()
	clone.Status.AttachError = &storage.VolumeError{
		Message: err.Error(),
		Time:    metav1.Now(),
	}
	newVa, err := h.client.StorageV1beta1().VolumeAttachments().Update(clone)
	if err != nil {
		return va, err
	}
	klog.V(4).Infof("Saved attach error to %q", va.Name)
	return newVa, nil
}

func (h *csiHandler) saveDetachError(va *storage.VolumeAttachment, err error) (*storage.VolumeAttachment, error) {
	klog.V(4).Infof("Saving detach error to %q", va.Name)
	clone := va.DeepCopy()
	clone.Status.DetachError = &storage.VolumeError{
		Message: err.Error(),
		Time:    metav1.Now(),
	}
	newVa, err := h.client.StorageV1beta1().VolumeAttachments().Update(clone)
	if err != nil {
		return va, err
	}
	klog.V(4).Infof("Saved detach error to %q", va.Name)
	return newVa, nil
}

func (h *csiHandler) SyncNewOrUpdatedPersistentVolume(pv *v1.PersistentVolume) {
	klog.V(4).Infof("CSIHandler: processing PV %q", pv.Name)
	// Sync and remove finalizer on given PV
	if pv.DeletionTimestamp == nil {
		// Don't process anything that has no deletion timestamp.
		klog.V(4).Infof("CSIHandler: processing PV %q: no deletion timestamp, ignoring", pv.Name)
		h.pvQueue.Forget(pv.Name)
		return
	}

	// Check if the PV has finalizer
	finalizer := GetFinalizerName(h.attacherName)
	found := false
	for _, f := range pv.Finalizers {
		if f == finalizer {
			found = true
			break
		}
	}
	if !found {
		// No finalizer -> no action required
		klog.V(4).Infof("CSIHandler: processing PV %q: no finalizer, ignoring", pv.Name)
		h.pvQueue.Forget(pv.Name)
		return
	}

	// Check that there is no VA that requires the PV
	vas, err := h.vaLister.List(labels.Everything())
	if err != nil {
		// Failed listing VAs? Try again with exp. backoff
		klog.Errorf("Failed to list VolumeAttachments for PV %q: %s", pv.Name, err.Error())
		h.pvQueue.AddRateLimited(pv.Name)
		return
	}
	for _, va := range vas {
		if va.Spec.Source.PersistentVolumeName != nil && *va.Spec.Source.PersistentVolumeName == pv.Name {
			// This PV is needed by this VA, don't remove finalizer
			klog.V(4).Infof("CSIHandler: processing PV %q: VA %q found", pv.Name, va.Name)
			h.pvQueue.Forget(pv.Name)
			return
		}
	}
	// No VA found -> remove finalizer
	klog.V(4).Infof("CSIHandler: processing PV %q: no VA found, removing finalizer", pv.Name)
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
		klog.Errorf("Failed to remove finalizer from PV %q: %s", pv.Name, err.Error())
		h.pvQueue.AddRateLimited(pv.Name)
		return
	}
	klog.V(2).Infof("Removed finalizer from PV %q", pv.Name)
	h.pvQueue.Forget(pv.Name)

	return
}

func (h *csiHandler) getCredentialsFromPV(csiSource *v1.CSIPersistentVolumeSource) (map[string]string, error) {
	if csiSource == nil {
		return nil, fmt.Errorf("CSI volume source was nil")
	}
	secretRef := csiSource.ControllerPublishSecretRef
	if secretRef == nil {
		return nil, nil
	}

	secret, err := h.client.CoreV1().Secrets(secretRef.Namespace).Get(secretRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load secret \"%s/%s\": %s", secretRef.Namespace, secretRef.Name, err)
	}
	credentials := map[string]string{}
	for key, value := range secret.Data {
		credentials[key] = string(value)
	}

	return credentials, nil
}

// getNodeID finds node ID from Node API object. If caller wants, it can find
// node ID stored in VolumeAttachment annotation.
func (h *csiHandler) getNodeID(driver string, nodeName string, va *storage.VolumeAttachment) (string, error) {
	// Try to find CSINodeInfo first.
	// nodeInfo, err := h.nodeInfoLister.Get(nodeName) // TODO (kubernetes/kubernetes #71052) use the lister once it syncs existing CSINodeInfo objects properly.
	nodeInfo, err := h.csiClientSet.CsiV1alpha1().CSINodeInfos().Get(nodeName, metav1.GetOptions{})
	if err == nil {
		if nodeID, found := GetNodeIDFromNodeInfo(driver, nodeInfo); found {
			klog.V(4).Infof("Found NodeID %s in CSINodeInfo %s", nodeID, nodeName)
			return nodeID, nil
		}
		klog.V(4).Infof("CSINodeInfo %s does not contain driver %s", nodeName, driver)
		// CSINodeInfo exists, but does not have the requested driver.
		// Fall through to Node annotation.
	} else {
		// Can't get CSINodeInfo, fall through to Node annotation.
		klog.V(4).Infof("Can't get CSINodeInfo %s: %s", nodeName, err)
	}

	// Check Node annotation.
	node, err := h.nodeLister.Get(nodeName)
	if err == nil {
		return GetNodeIDFromNode(driver, node)
	}

	// Check VolumeAttachment annotation as the last resort if caller wants so (i.e. has provided one).
	if va == nil {
		return "", err
	}
	if nodeID, found := va.Annotations[vaNodeIDAnnotation]; found {
		return nodeID, nil
	}

	// return nodeLister.Get error
	return "", err
}
