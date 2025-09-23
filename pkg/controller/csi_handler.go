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
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/external-attacher/pkg/attacher"
	"github.com/kubernetes-csi/external-attacher/pkg/features"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type AttacherCSITranslator interface {
	TranslateInTreePVToCSI(logger klog.Logger, pv *v1.PersistentVolume) (*v1.PersistentVolume, error)
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
	vaQueue, pvQueue              workqueue.TypedRateLimitingInterface[string]
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

func (h *csiHandler) Init(vaQueue workqueue.TypedRateLimitingInterface[string], pvQueue workqueue.TypedRateLimitingInterface[string]) {
	h.vaQueue = vaQueue
	h.pvQueue = pvQueue
}

// ReconcileVA lists volumes from the CSI Driver and reconciles the attachment
// status with the corresponding VolumeAttachment object. If the attachment
// status of the volume is different from the state on the VolumeAttachment the
// VolumeAttachment object is patched to the correct state.
func (h *csiHandler) ReconcileVA(ctx context.Context) error {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Reconciling VolumeAttachments with driver backend state")

	ctx, cancel := context.WithTimeout(ctx, h.timeout)
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
		if va.Spec.Attacher != h.attacherName {
			// skip VolumeAttachments of other CSI drivers
			continue
		}

		nodeID, err := h.getNodeID(logger, h.attacherName, va.Spec.NodeName, va)
		if err != nil {
			logger.Error(err, "Failed to find node ID err")
			continue
		}
		pvSpec, err := h.getProcessedPVSpec(ctx, va)
		if err != nil {
			logger.Error(err, "Failed to get PV Spec")
			continue
		}

		source, err := getCSISource(pvSpec)
		if err != nil {
			logger.Error(err, "Failed to get CSI Source")
			continue
		}

		volumeHandle, _, err := GetVolumeHandle(source)
		if err != nil {
			logger.Error(err, "Failed to get volume handle")
			continue
		}
		attachedStatus := va.Status.Attached

		// If volume driver has corresponding in-tree plugin, generate a correct volumehandle
		isMig, err := h.isMigratable(va)
		if err != nil {
			logger.Error(err, "Failed to check if migratable for volume handle", "volumeHandle", volumeHandle, "sourceDriver", source.Driver)
			continue
		}
		if isMig {
			volumeHandle, err = h.translator.RepairVolumeHandle(source.Driver, volumeHandle, nodeID)
			if err != nil {
				logger.Error(err, "Failed to repair volume handle for driver", "volumeHandle", volumeHandle, "sourceDriver", source.Driver)
				continue
			}
		}

		// Check whether the volume is published to this node
		found := slices.Contains(published[volumeHandle], nodeID)

		// If ListVolumes Attached Status is different, add to shared workQueue.
		if attachedStatus != found {
			logger.Error(
				nil,
				"VolumeAttachment attached status and actual state do not match. Adding back to VolumeAttachment queue for forced reprocessing",
				"VolumeAttachment", va.Name,
				"volumeHandle", volumeHandle,
				"attachedStatus", attachedStatus,
				"found", found,
			)
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

func (h *csiHandler) SyncNewOrUpdatedVolumeAttachment(ctx context.Context, va *storage.VolumeAttachment) {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("CSIHandler: processing VolumeAttachment")

	var err error
	if va.DeletionTimestamp == nil {
		err = h.syncAttach(ctx, va)
	} else {
		err = h.syncDetach(ctx, va)
	}
	if err != nil {
		// Re-queue with exponential backoff
		logger.V(2).Info("Error processing", "err", err)
		h.vaQueue.AddRateLimited(va.Name)
		return
	}
	// The operation has finished successfully, reset exponential backoff
	h.vaQueue.Forget(va.Name)
	logger.V(4).Info("CSIHandler: finished processing")
}

func (h *csiHandler) syncAttach(ctx context.Context, va *storage.VolumeAttachment) error {
	logger := klog.FromContext(ctx)
	if !h.consumeForceSync(va.Name) && va.Status.Attached {
		// Volume is attached and no force sync, there is nothing to be done.
		logger.V(4).Info("VolumeAttachment is already attached")
		return nil
	}

	// Attach and report any error
	logger.V(2).Info("Attaching")
	va, metadata, err := h.csiAttach(ctx, va)
	if err != nil {
		_, saveErr := h.saveAttachError(ctx, va, err)
		if saveErr != nil {
			// Just log it, propagate the attach error.
			logger.V(2).Info("Failed to save attach error to VolumeAttachment", "err", saveErr.Error())
		}
		// Add context to the error for logging
		err := fmt.Errorf("failed to attach: %s", err)
		return err
	}
	logger.V(2).Info("Attached")

	// Mark as attached
	if _, err := markAsAttached(ctx, h.client, va, metadata); err != nil {
		return fmt.Errorf("failed to mark as attached: %s", err)
	}
	logger.V(4).Info("Fully attached")
	return nil
}

func (h *csiHandler) syncDetach(ctx context.Context, va *storage.VolumeAttachment) error {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Starting detach operation")
	if !h.consumeForceSync(va.Name) && !h.hasVAFinalizer(va) {
		logger.V(4).Info("VolumeAttachment is already detached")
		return nil
	}

	// Detach and report any error
	logger.V(2).Info("Detaching")
	va, err := h.csiDetach(ctx, va)
	if err != nil {
		var saveErr error
		_, saveErr = h.saveDetachError(ctx, va, err)
		if saveErr != nil {
			// Just log it, propagate the detach error.
			logger.V(2).Info("Failed to save detach error to VolumeAttachment", "err", saveErr.Error())
		}
		// Add context to the error for logging
		err := fmt.Errorf("failed to detach: %s", err)
		return err
	}
	logger.V(4).Info("Fully detached")
	return nil
}

func (h *csiHandler) prepareVAFinalizer(logger klog.Logger, va *storage.VolumeAttachment) (newVA *storage.VolumeAttachment, modified bool) {
	finalizerName := GetFinalizerName(h.attacherName)
	if slices.Contains(va.Finalizers, finalizerName) {
		// Finalizer is already present
		logger.V(4).Info("VolumeAttachment finalizer is already set")
		return va, false
	}

	// Finalizer is not present, add it
	clone := va.DeepCopy()
	clone.Finalizers = append(clone.Finalizers, finalizerName)
	logger.V(4).Info("VolumeAttachment finalizer added")
	return clone, true
}

func (h *csiHandler) prepareVANodeID(logger klog.Logger, va *storage.VolumeAttachment, nodeID string) (newVA *storage.VolumeAttachment, modified bool) {
	logger = klog.LoggerWithValues(logger, "nodeID", nodeID)
	if existingID, ok := va.Annotations[vaNodeIDAnnotation]; ok && existingID == nodeID {
		logger.V(4).Info("NodeID annotation is already set")
		return va, false
	}
	clone := va.DeepCopy()
	if clone.Annotations == nil {
		clone.Annotations = map[string]string{}
	}
	clone.Annotations[vaNodeIDAnnotation] = nodeID
	logger.V(4).Info("NodeID annotation added")
	return clone, true
}

func (h *csiHandler) addPVFinalizer(ctx context.Context, pv *v1.PersistentVolume) (*v1.PersistentVolume, error) {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "PersistentVolume", pv.Name)
	finalizerName := GetFinalizerName(h.attacherName)
	if slices.Contains(pv.Finalizers, finalizerName) {
		// Finalizer is already present
		logger.V(4).Info("PersistentVolume finalizer is already set")
		return pv, nil
	}

	// Finalizer is not present, add it
	logger.V(4).Info("Adding finalizer to PersistentVolume")
	clone := pv.DeepCopy()
	clone.Finalizers = append(clone.Finalizers, finalizerName)

	newPV, err := h.patchPV(ctx, pv, clone)
	if err != nil {
		return pv, err
	}

	logger.V(4).Info("PersistentVolume finalizer added")
	return newPV, nil
}

func (h *csiHandler) hasVAFinalizer(va *storage.VolumeAttachment) bool {
	finalizerName := GetFinalizerName(h.attacherName)
	return slices.Contains(va.Finalizers, finalizerName)
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

func (h *csiHandler) getProcessedPVSpec(ctx context.Context, va *storage.VolumeAttachment) (*v1.PersistentVolumeSpec, error) {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Starting processing PVSpec operation")

	if va.Spec.Source.PersistentVolumeName != nil {
		if va.Spec.Source.InlineVolumeSpec != nil {
			return nil, errors.New("both InlineCSIVolumeSource and PersistentVolumeName specified in VA source")
		}
		pv, err := h.pvLister.Get(*va.Spec.Source.PersistentVolumeName)
		if err != nil {
			return nil, err
		}
		if h.translator.IsPVMigratable(pv) {
			pv, err = h.translator.TranslateInTreePVToCSI(logger, pv)
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

func (h *csiHandler) csiAttach(ctx context.Context, va *storage.VolumeAttachment) (*storage.VolumeAttachment, map[string]string, error) {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Starting attach operation")
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
		pv, err = h.addPVFinalizer(ctx, pv)
		if err != nil {
			return va, nil, fmt.Errorf("could not add PersistentVolume finalizer: %s", err)
		}

		if h.translator.IsPVMigratable(pv) {
			pv, err = h.translator.TranslateInTreePVToCSI(logger, pv)
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

	volumeCapabilities, err := GetVolumeCapabilities(logger, pvSpec, h.supportsSingleNodeMultiWriter, h.defaultFSType)
	if err != nil {
		return va, nil, err
	}

	secrets, err := h.getCredentialsFromPV(ctx, csiSource)
	if err != nil {
		return va, nil, err
	}

	nodeID, err := h.getNodeID(logger, h.attacherName, va.Spec.NodeName, nil)
	if err != nil {
		return va, nil, err
	}

	originalVA := va
	va, finalizerAdded := h.prepareVAFinalizer(logger, va)
	va, nodeIDAdded := h.prepareVANodeID(logger, va, nodeID)

	if finalizerAdded || nodeIDAdded {
		if va, err = h.patchVA(ctx, originalVA, va); err != nil {
			return originalVA, nil, fmt.Errorf("could not save VolumeAttachment: %s", err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, h.timeout)
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

func (h *csiHandler) csiDetach(ctx context.Context, va *storage.VolumeAttachment) (*storage.VolumeAttachment, error) {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Starting detach operation")

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
			pv, err = h.translator.TranslateInTreePVToCSI(logger, pv)
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
	secrets, err := h.getCredentialsFromPV(ctx, csiSource)
	if err != nil {
		return va, err
	}

	nodeID, err := h.getNodeID(logger, h.attacherName, va.Spec.NodeName, va)
	if err != nil {
		return va, err
	}

	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	ctx = markAsMigrated(ctx, migratable)
	defer cancel()
	err = h.attacher.Detach(ctx, volumeHandle, nodeID, secrets)
	if err != nil {
		// The volume may not be fully detached. Save the error and try again
		// after backoff.
		return va, err
	}
	logger.V(2).Info("Detached")

	if va, err := markAsDetached(ctx, h.client, va); err != nil {
		return va, fmt.Errorf("could not mark as detached: %s", err)
	}

	return va, nil
}

func (h *csiHandler) saveAttachError(ctx context.Context, va *storage.VolumeAttachment, err error) (*storage.VolumeAttachment, error) {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Saving attach error")
	clone := va.DeepCopy()

	volumeError := &storage.VolumeError{
		Message: err.Error(),
		Time:    metav1.Now(),
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.MutableCSINodeAllocatableCount) {
		if st, ok := status.FromError(err); ok {
			errorCode := int32(st.Code())
			volumeError.ErrorCode = &errorCode
		}
	}

	clone.Status.AttachError = volumeError

	var newVa *storage.VolumeAttachment
	if newVa, err = h.patchVA(ctx, va, clone, "status"); err != nil {
		return va, err
	}
	logger.V(4).Info("Saved attach error")
	return newVa, nil
}

func (h *csiHandler) saveDetachError(ctx context.Context, va *storage.VolumeAttachment, err error) (*storage.VolumeAttachment, error) {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Saving detach error")
	clone := va.DeepCopy()
	clone.Status.DetachError = &storage.VolumeError{
		Message: err.Error(),
		Time:    metav1.Now(),
	}

	var newVa *storage.VolumeAttachment
	if newVa, err = h.patchVA(ctx, va, clone, "status"); err != nil {
		return va, err
	}
	logger.V(4).Info("Saved detach error")
	return newVa, nil
}

func (h *csiHandler) SyncNewOrUpdatedPersistentVolume(ctx context.Context, pv *v1.PersistentVolume) {
	logger := klog.FromContext(ctx)
	logger.V(4).Info("CSIHandler: processing PersistentVolume")
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
					logger.V(4).Info(
						"CSIHandler: PersistentVolume is an in-tree PV but does not have migrated-to annotation or the annotation does not match. Remove the finalizer for this PersistentVolume",
						"expect", h.attacherName,
						"get", migratedToDriver,
					)
				}
			}
		}

		if ignore {
			// Don't process anything that has no deletion timestamp.
			logger.V(4).Info("CSIHandler: processing PersistentVolume: no deletion timestamp, ignoring")
			h.pvQueue.Forget(pv.Name)
			return
		}
	}

	// Check if the PV has finalizer
	finalizer := GetFinalizerName(h.attacherName)
	found := slices.Contains(pv.Finalizers, finalizer)
	if !found {
		// No finalizer -> no action required
		logger.V(4).Info("CSIHandler: processing PersistentVolume: no finalizer, ignoring")
		h.pvQueue.Forget(pv.Name)
		return
	}

	// Check that there is no VA that requires the PV
	vas, err := h.vaLister.List(labels.Everything())
	if err != nil {
		// Failed listing VAs? Try again with exp. backoff
		logger.Error(err, "Failed to list VolumeAttachments for PersistentVolume")
		h.pvQueue.AddRateLimited(pv.Name)
		return
	}
	for _, va := range vas {
		if va.Spec.Source.PersistentVolumeName != nil && *va.Spec.Source.PersistentVolumeName == pv.Name {
			// This PV is needed by this VA, don't remove finalizer
			logger.V(4).Info("CSIHandler: processing PersistentVolume: VolumeAttachment was found", "VolumeAttachment", va.Name)
			h.pvQueue.Forget(pv.Name)
			return
		}
	}
	// No VA found -> remove finalizer
	logger.V(4).Info("CSIHandler: processing PersistentVolume: no VolumeAttachment found, removing finalizer")
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

	if _, err = h.patchPV(ctx, pv, clone); err != nil {
		logger.Error(err, "Failed to remove finalizer from PersistentVolume")
		h.pvQueue.AddRateLimited(pv.Name)
		return
	}

	logger.V(2).Info("Removed finalizer from PersistentVolume")
	h.pvQueue.Forget(pv.Name)
}

func (h *csiHandler) getCredentialsFromPV(ctx context.Context, csiSource *v1.CSIPersistentVolumeSource) (map[string]string, error) {
	if csiSource == nil {
		return nil, fmt.Errorf("CSI volume source was nil")
	}
	secretRef := csiSource.ControllerPublishSecretRef
	if secretRef == nil {
		return nil, nil
	}

	secret, err := h.client.CoreV1().Secrets(secretRef.Namespace).Get(ctx, secretRef.Name, metav1.GetOptions{})
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
func (h *csiHandler) getNodeID(logger klog.Logger, driver string, nodeName string, va *storage.VolumeAttachment) (string, error) {
	// Try to find CSINode first.
	csiNode, err := h.csiNodeLister.Get(nodeName)
	if err == nil {
		if nodeID, found := GetNodeIDFromCSINode(driver, csiNode); found {
			logger.V(4).Info("Found nodeID in CSINode", "nodeID", nodeID, "CSINode", nodeName)
			return nodeID, nil
		}
		// CSINode exists, but does not have the requested driver; this can happen if the CSI pod is not running, for
		// example the node might be currently shut down. We don't want to block the controller unpublish in that scenario.
		// We should treat missing driver in the same way as missing CSINode; attempt to use the node ID from the volume
		// attachment.
		err = fmt.Errorf("CSINode %s does not contain driver %s", nodeName, driver)
	}

	// Can't get CSINode, check Volume Attachment.
	logger.V(4).Info("Failed to get nodeID from CSINode", "nodeName", nodeName, "err", err.Error())

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

func (h *csiHandler) patchVA(ctx context.Context, va, clone *storage.VolumeAttachment, subresources ...string) (*storage.VolumeAttachment,
	error) {
	patch, err := createMergePatch(va, clone)
	if err != nil {
		return va, err
	}

	newVa, err := h.client.StorageV1().VolumeAttachments().Patch(ctx, va.Name, types.MergePatchType, patch, metav1.PatchOptions{}, subresources...)
	if err != nil {
		return va, err
	}
	return newVa, nil
}

func (h *csiHandler) patchPV(ctx context.Context, pv, clone *v1.PersistentVolume) (*v1.PersistentVolume, error) {
	patch, err := createMergePatch(pv, clone)
	if err != nil {
		return pv, err
	}

	newPV, err := h.client.CoreV1().PersistentVolumes().Patch(ctx, pv.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return pv, err
	}
	return newPV, nil
}

func markAsMigrated(parent context.Context, hasMigrated bool) context.Context {
	return context.WithValue(parent, connection.AdditionalInfoKey, connection.AdditionalInfo{Migrated: strconv.FormatBool(hasMigrated)})
}
