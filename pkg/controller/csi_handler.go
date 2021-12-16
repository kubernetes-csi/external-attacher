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
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/external-attacher/pkg/attacher"
	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type AttacherCSITranslator interface {
	TranslateInTreePVToCSI(pv *v1.PersistentVolume) (*v1.PersistentVolume, error)
	IsPVMigratable(pv *v1.PersistentVolume) bool
	RepairVolumeHandle(pluginName, volumeHandle, nodeID string) (string, error)
}

// Lister implements list operations against a remote CSI driver.
type VolumeLister interface {
	// ListVolumes calls ListVolumes on the driver and returns a map with keys
	// of VolumeID and values of the list of Node IDs that volume is published
	// on
	ListVolumes(ctx context.Context) (map[string][]string, error)
}

var _ VolumeLister = &attacher.CSIVolumeLister{}

// csiHandler is a handler that calls CSI to attach/detach volume.
// It adds finalizer to VolumeAttachment instance to make sure they're detached
// before deletion.
type csiHandler struct {
	client                        kubernetes.Interface
	attacherName                  string
	attacher                      attacher.Attacher
	CSIVolumeLister               VolumeLister
	pvLister                      corelisters.PersistentVolumeLister
	csiNodeLister                 storagelisters.CSINodeLister
	vaLister                      storagelisters.VolumeAttachmentLister
	vaQueue, pvQueue              workqueue.RateLimitingInterface
	forceSync                     map[string]bool
	forceSyncMux                  sync.Mutex
	timeout                       time.Duration
	supportsPublishReadOnly       bool
	supportsSingleNodeMultiWriter bool
	translator                    AttacherCSITranslator
	defaultFSType                 string
}

var _ Handler = &csiHandler{}

// NewCSIHandler creates a new CSIHandler.
func NewCSIHandler(
	client kubernetes.Interface,
	attacherName string,
	attacher attacher.Attacher,
	CSIVolumeLister VolumeLister,
	pvLister corelisters.PersistentVolumeLister,
	csiNodeLister storagelisters.CSINodeLister,
	vaLister storagelisters.VolumeAttachmentLister,
	timeout *time.Duration,
	supportsPublishReadOnly bool,
	supportsSingleNodeMultiWriter bool,
	translator AttacherCSITranslator,
	defaultFSType string) Handler {

	return &csiHandler{
		client:                        client,
		attacherName:                  attacherName,
		attacher:                      attacher,
		CSIVolumeLister:               CSIVolumeLister,
		pvLister:                      pvLister,
		csiNodeLister:                 csiNodeLister,
		vaLister:                      vaLister,
		timeout:                       *timeout,
		supportsPublishReadOnly:       supportsPublishReadOnly,
		supportsSingleNodeMultiWriter: supportsSingleNodeMultiWriter,
		translator:                    translator,
		forceSync:                     map[string]bool{},
		forceSyncMux:                  sync.Mutex{},
		defaultFSType:                 defaultFSType,
	}
}

func (h *csiHandler) Init(vaQueue workqueue.RateLimitingInterface, pvQueue workqueue.RateLimitingInterface) {
	h.vaQueue = vaQueue
	h.pvQueue = pvQueue
}

// ReconcileVA lists volumes from the CSI Driver and reconciles the attachment
// status with the corresponding VolumeAttachment object. If the attachment
// status of the volume is different from the state on the VolumeAttachment the
// VolumeAttachment object is patched to the correct state.
func (h *csiHandler) ReconcileVA() error {
	klog.V(4).Info("Reconciling VolumeAttachments with driver backend state")

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// Loop over all volume attachment objects
	vas, err := h.vaLister.List(labels.Everything())
	if err != nil {
		return errors.New("failed to list all VolumeAttachment objects")
	}

	published, err := h.CSIVolumeLister.ListVolumes(ctx)
	if err != nil {
		return fmt.Errorf("failed to ListVolumes: %v", err)
	}

	for _, va := range vas {
		nodeID, err := h.getNodeID(h.attacherName, va.Spec.NodeName, va)
		if err != nil {
			klog.Warningf("Failed to find node ID err: %v", err)
			continue
		}
		pvSpec, err := h.getProcessedPVSpec(va)
		if err != nil {
			klog.Warningf("Failed to get PV Spec: %v", err)
			continue
		}

		source, err := getCSISource(pvSpec)
		if err != nil {
			klog.Warningf("Failed to get CSI Source: %v", err)
			continue
		}

		volumeHandle, _, err := GetVolumeHandle(source)
		if err != nil {
			klog.Warningf("Failed to get volume handle: %v", err)
			continue
		}
		attachedStatus := va.Status.Attached

		// If volume driver has corresponding in-tree plugin, generate a correct volumehandle
		isMig, err := h.isMigratable(va)
		if err != nil {
			klog.Warningf("Failed to check if migratable for volume handle %s (driver %s): %v", volumeHandle, source.Driver, err)
			continue
		}
		if isMig {
			volumeHandle, err = h.translator.RepairVolumeHandle(source.Driver, volumeHandle, nodeID)
			if err != nil {
				klog.Warningf("Failed to repair volume handle %s for driver %s: %v", volumeHandle, source.Driver, err)
				continue
			}
		}

		// Check whether the volume is published to this node
		found := false
		for _, gotNodeID := range published[volumeHandle] {
			if gotNodeID == nodeID {
				found = true
				break
			}
		}

		// If ListVolumes Attached Status is different, add to shared workQueue.
		if attachedStatus != found {
			klog.Warningf("VA %s for volume %s has attached status %v but actual state %v. Adding back to VA queue for forced reprocessing", va.Name, volumeHandle, attachedStatus, found)
			// Add this item to the vaQueue with forceSync so that it is force
			// processed again, we avoid UPDATE on the VA or forcing a direct
			// attach/detach as to avoid race conditions with the main attacher
			// queue
			h.setForceSync(va.Name)
			h.vaQueue.Add(va.Name)
		}
	}
	return nil
}

// setForceSync sets the intention that next time the VolumeAttachment
// referenced by vaName is processed on the VA queue that attach or detach will
// proceed even when the VA.Status.Attached may already show the desired state
func (h *csiHandler) setForceSync(vaName string) {
	h.forceSyncMux.Lock()
	defer h.forceSyncMux.Unlock()
	h.forceSync[vaName] = true
}

// consumeForceSync is used to check whether forceSync was set for the VA
// referenced by vaName. It will then remove the forceSync intention so that the
// VA will only be forceSync-ed once per request
func (h *csiHandler) consumeForceSync(vaName string) bool {
	h.forceSyncMux.Lock()
	defer h.forceSyncMux.Unlock()
	s, ok := h.forceSync[vaName]
	if ok {
		delete(h.forceSync, vaName)
	}
	return s
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
	if !h.consumeForceSync(va.Name) && va.Status.Attached {
		// Volume is attached and no force sync, there is nothing to be done.
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
	if !h.consumeForceSync(va.Name) && !h.hasVAFinalizer(va) {
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

	newPV, err := h.patchPV(pv, clone)
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

// Checks if the PV (or) the inline-volume corresponding to the VA could have migrated from
// in-tree to CSI.
func (h *csiHandler) isMigratable(va *storage.VolumeAttachment) (bool, error) {
	if va.Spec.Source.PersistentVolumeName != nil {
		pv, err := h.pvLister.Get(*va.Spec.Source.PersistentVolumeName)
		if err != nil {
			return false, err
		}
		return h.translator.IsPVMigratable(pv), nil
	} else if va.Spec.Source.InlineVolumeSpec != nil {
		if va.Spec.Source.InlineVolumeSpec.CSI == nil {
			return false, errors.New("inline volume spec contains nil CSI source")
		}
		return true, nil
	} else {
		return false, nil
	}
}

func getCSISource(pvSpec *v1.PersistentVolumeSpec) (*v1.CSIPersistentVolumeSource, error) {
	if pvSpec == nil {
		return nil, errors.New("could not get CSI source, pv spec was nil")
	}
	if pvSpec.CSI != nil {
		return pvSpec.CSI, nil
	}
	return nil, errors.New("pv spec contained non-csi source that was not migrated")
}

func (h *csiHandler) getProcessedPVSpec(va *storage.VolumeAttachment) (*v1.PersistentVolumeSpec, error) {
	if va.Spec.Source.PersistentVolumeName != nil {
		if va.Spec.Source.InlineVolumeSpec != nil {
			return nil, errors.New("both InlineCSIVolumeSource and PersistentVolumeName specified in VA source")
		}
		pv, err := h.pvLister.Get(*va.Spec.Source.PersistentVolumeName)
		if err != nil {
			return nil, err
		}
		if h.translator.IsPVMigratable(pv) {
			pv, err = h.translator.TranslateInTreePVToCSI(pv)
			if err != nil {
				return nil, fmt.Errorf("failed to TranslateInTreePVToCSI(%v): %v", pv, err)
			}
		}
		return &pv.Spec, nil
	} else if va.Spec.Source.InlineVolumeSpec != nil {
		if va.Spec.Source.InlineVolumeSpec.CSI == nil {
			return nil, errors.New("inline volume spec contains nil CSI source")
		}

		return va.Spec.Source.InlineVolumeSpec, nil
	} else {
		return nil, errors.New("neither InlineCSIVolumeSource nor PersistentVolumeName specified in VA source")
	}
}

func (h *csiHandler) csiAttach(va *storage.VolumeAttachment) (*storage.VolumeAttachment, map[string]string, error) {
	klog.V(4).Infof("Starting attach operation for %q", va.Name)
	// Check as much as possible before adding VA finalizer - it would block
	// deletion of VA on error.

	var csiSource *v1.CSIPersistentVolumeSource
	var pvSpec *v1.PersistentVolumeSpec
	var migratable bool
	if va.Spec.Source.PersistentVolumeName != nil {
		if va.Spec.Source.InlineVolumeSpec != nil {
			return va, nil, errors.New("both InlineCSIVolumeSource and PersistentVolumeName specified in VA source")
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

		if h.translator.IsPVMigratable(pv) {
			pv, err = h.translator.TranslateInTreePVToCSI(pv)
			if err != nil {
				return va, nil, fmt.Errorf("failed to translate in tree pv to CSI: %v", err)
			}
			migratable = true
		}

		// Both csiSource and pvSpec could be translated here if the PV was
		// migrated
		csiSource, err = getCSISource(&pv.Spec)
		if err != nil {
			return va, nil, err
		}

		pvSpec = &pv.Spec
	} else if va.Spec.Source.InlineVolumeSpec != nil {
		if va.Spec.Source.InlineVolumeSpec.CSI != nil {
			csiSource = va.Spec.Source.InlineVolumeSpec.CSI
		} else {
			return va, nil, errors.New("inline volume spec contains nil CSI source")
		}

		pvSpec = va.Spec.Source.InlineVolumeSpec
	} else {
		return va, nil, errors.New("neither InlineCSIVolumeSource nor PersistentVolumeName specified in VA source")
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

	volumeCapabilities, err := GetVolumeCapabilities(pvSpec, h.supportsSingleNodeMultiWriter, h.defaultFSType)
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
		if va, err = h.patchVA(originalVA, va); err != nil {
			return originalVA, nil, fmt.Errorf("could not save VolumeAttachment: %s", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	ctx = markAsMigrated(ctx, migratable)
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
	var csiSource *v1.CSIPersistentVolumeSource
	var migratable bool
	if va.Spec.Source.PersistentVolumeName != nil {
		if va.Spec.Source.InlineVolumeSpec != nil {
			return va, errors.New("both InlineCSIVolumeSource and PersistentVolumeName specified in VA source")
		}
		pv, err := h.pvLister.Get(*va.Spec.Source.PersistentVolumeName)
		if err != nil {
			return va, err
		}
		if h.translator.IsPVMigratable(pv) {
			pv, err = h.translator.TranslateInTreePVToCSI(pv)
			if err != nil {
				return va, fmt.Errorf("failed to translate in tree pv to CSI: %v", err)
			}
			migratable = true
		}
		csiSource, err = getCSISource(&pv.Spec)
		if err != nil {
			return va, err
		}
	} else if va.Spec.Source.InlineVolumeSpec != nil {
		if va.Spec.Source.InlineVolumeSpec.CSI != nil {
			csiSource = va.Spec.Source.InlineVolumeSpec.CSI
		} else {
			return va, errors.New("inline volume spec contains nil CSI source")
		}
	} else {
		return va, errors.New("neither InlineCSIVolumeSource nor PersistentVolumeName specified in VA source")
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
	ctx = markAsMigrated(ctx, migratable)
	defer cancel()
	err = h.attacher.Detach(ctx, volumeHandle, nodeID, secrets)
	if err != nil {
		// The volume may not be fully detached. Save the error and try again
		// after backoff.
		return va, err
	}
	klog.V(2).Infof("Detached %q", va.Name)

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

	var newVa *storage.VolumeAttachment
	if newVa, err = h.patchVA(va, clone, "status"); err != nil {
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

	var newVa *storage.VolumeAttachment
	if newVa, err = h.patchVA(va, clone, "status"); err != nil {
		return va, err
	}
	klog.V(4).Infof("Saved detach error to %q", va.Name)
	return newVa, nil
}

func (h *csiHandler) SyncNewOrUpdatedPersistentVolume(pv *v1.PersistentVolume) {
	klog.V(4).Infof("CSIHandler: processing PV %q", pv.Name)
	// Sync and remove finalizer on given PV
	if pv.DeletionTimestamp == nil {
		ignore := true

		// if the PV is migrated this means CSIMigration is disabled so we need to remove the finalizer
		// to give back the control of the PV to Kube-Controller-Manager
		if h.translator.IsPVMigratable(pv) {
			ignore = false
			if ann := pv.Annotations; ann != nil {
				if migratedToDriver := ann[annMigratedTo]; migratedToDriver == h.attacherName {
					ignore = true
				} else {
					klog.V(4).Infof("CSIHandler: PV %q is an in-tree PV but does not have migrated-to annotation "+
						"or the annotation does not match. Expect %v, Get %v. Remove the finalizer for this PV ",
						pv.Name, h.attacherName, migratedToDriver)
				}
			}
		}

		if ignore {
			// Don't process anything that has no deletion timestamp.
			klog.V(4).Infof("CSIHandler: processing PV %q: no deletion timestamp, ignoring", pv.Name)
			h.pvQueue.Forget(pv.Name)
			return
		}
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

	if _, err = h.patchPV(pv, clone); err != nil {
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

	secret, err := h.client.CoreV1().Secrets(secretRef.Namespace).Get(context.TODO(), secretRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load secret \"%s/%s\": %s", secretRef.Namespace, secretRef.Name, err)
	}
	credentials := map[string]string{}
	for key, value := range secret.Data {
		credentials[key] = string(value)
	}

	return credentials, nil
}

// getNodeID finds node ID from CSINode API object. If caller wants, it can find
// node ID stored in VolumeAttachment annotation.
func (h *csiHandler) getNodeID(driver string, nodeName string, va *storage.VolumeAttachment) (string, error) {
	// Try to find CSINode first.
	csiNode, err := h.csiNodeLister.Get(nodeName)
	if err == nil {
		if nodeID, found := GetNodeIDFromCSINode(driver, csiNode); found {
			klog.V(4).Infof("Found NodeID %s in CSINode %s", nodeID, nodeName)
			return nodeID, nil
		}
		// CSINode exists, but does not have the requested driver; this can happen if the CSI pod is not running, for
		// example the node might be currently shut down. We don't want to block the controller unpublish in that scenario.
		// We should treat missing driver in the same way as missing CSINode; attempt to use the node ID from the volume
		// attachment.
		err = errors.New(fmt.Sprintf("CSINode %s does not contain driver %s", nodeName, driver))
	}

	// Can't get CSINode, check Volume Attachment.
	klog.V(4).Infof("Can't get nodeID from CSINode %s: %s", nodeName, err)

	// Check VolumeAttachment annotation as the last resort if caller wants so (i.e. has provided one).
	if va == nil {
		return "", err
	}
	if nodeID, found := va.Annotations[vaNodeIDAnnotation]; found {
		return nodeID, nil
	}

	// return csiNodeLister.Get error
	return "", err
}

func (h *csiHandler) patchVA(va, clone *storage.VolumeAttachment, subresources ...string) (*storage.VolumeAttachment,
	error) {
	patch, err := createMergePatch(va, clone)
	if err != nil {
		return va, err
	}

	newVa, err := h.client.StorageV1().VolumeAttachments().Patch(context.TODO(), va.Name, types.MergePatchType, patch,
		metav1.PatchOptions{}, subresources...)
	if err != nil {
		return va, err
	}
	return newVa, nil
}

func (h *csiHandler) patchPV(pv, clone *v1.PersistentVolume) (*v1.PersistentVolume, error) {
	patch, err := createMergePatch(pv, clone)
	if err != nil {
		return pv, err
	}

	newPV, err := h.client.CoreV1().PersistentVolumes().Patch(context.TODO(), pv.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return pv, err
	}
	return newPV, nil
}

func markAsMigrated(parent context.Context, hasMigrated bool) context.Context {
	return context.WithValue(parent, connection.AdditionalInfoKey, connection.AdditionalInfo{Migrated: strconv.FormatBool(hasMigrated)})
}
