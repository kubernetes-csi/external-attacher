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
	"sync"
	"time"

	"github.com/kubernetes-csi/external-attacher/pkg/features"
	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	coreinformers "k8s.io/client-go/informers/core/v1"
	storageinformers "k8s.io/client-go/informers/storage/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	csitrans "k8s.io/csi-translation-lib"
	"k8s.io/klog/v2"
)

const (
	annMigratedTo = "pv.kubernetes.io/migrated-to"
)

// CSIAttachController is a controller that attaches / detaches CSI volumes using provided Handler interface
type CSIAttachController struct {
	client       kubernetes.Interface
	attacherName string
	handler      Handler
	vaQueue      workqueue.TypedRateLimitingInterface[string]
	pvQueue      workqueue.TypedRateLimitingInterface[string]

	vaLister       storagelisters.VolumeAttachmentLister
	vaListerSynced cache.InformerSynced
	pvLister       corelisters.PersistentVolumeLister
	pvListerSynced cache.InformerSynced

	shouldReconcileVolumeAttachment bool
	reconcileSync                   time.Duration
	translator                      AttacherCSITranslator
}

// Handler is responsible for handling VolumeAttachment events from informer.
type Handler interface {
	Init(vaQueue workqueue.TypedRateLimitingInterface[string], pvQueue workqueue.TypedRateLimitingInterface[string])

	// SyncNewOrUpdatedVolumeAttachment processes one Add/Updated event from
	// VolumeAttachment informers. It runs in a workqueue, guaranting that only
	// one SyncNewOrUpdatedVolumeAttachment runs for given VA.
	// SyncNewOrUpdatedVolumeAttachment is responsible for marking the
	// VolumeAttachment either as forgotten (resets exponential backoff) or
	// re-queue it into the vaQueue to process it after exponential
	// backoff.
	SyncNewOrUpdatedVolumeAttachment(ctx context.Context, va *storage.VolumeAttachment)

	SyncNewOrUpdatedPersistentVolume(ctx context.Context, pv *v1.PersistentVolume)

	ReconcileVA(ctx context.Context) error
}

// NewCSIAttachController returns a new *CSIAttachController
func NewCSIAttachController(
	logger klog.Logger,
	client kubernetes.Interface,
	attacherName string,
	handler Handler,
	volumeAttachmentInformer storageinformers.VolumeAttachmentInformer,
	pvInformer coreinformers.PersistentVolumeInformer,
	vaRateLimiter, paRateLimiter workqueue.TypedRateLimiter[string],
	shouldReconcileVolumeAttachment bool,
	reconcileSync time.Duration,
) *CSIAttachController {
	ctrl := &CSIAttachController{
		client:                          client,
		attacherName:                    attacherName,
		handler:                         handler,
		vaQueue:                         workqueue.NewTypedRateLimitingQueueWithConfig(vaRateLimiter, workqueue.TypedRateLimitingQueueConfig[string]{Name: "csi-attacher-va"}),
		pvQueue:                         workqueue.NewTypedRateLimitingQueueWithConfig(paRateLimiter, workqueue.TypedRateLimitingQueueConfig[string]{Name: "csi-attacher-pv"}),
		shouldReconcileVolumeAttachment: shouldReconcileVolumeAttachment,
		reconcileSync:                   reconcileSync,
		translator:                      csitrans.New(),
	}

	volumeAttachmentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.vaAdded,
		UpdateFunc: ctrl.vaUpdatedFunc(logger),
		DeleteFunc: ctrl.vaDeleted,
	})
	ctrl.vaLister = volumeAttachmentInformer.Lister()
	ctrl.vaListerSynced = volumeAttachmentInformer.Informer().HasSynced

	pvInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.pvAdded,
		UpdateFunc: ctrl.pvUpdated,
		//DeleteFunc: ctrl.pvDeleted, TODO: do we need this?
	})
	ctrl.pvLister = pvInformer.Lister()
	ctrl.pvListerSynced = pvInformer.Informer().HasSynced
	ctrl.handler.Init(ctrl.vaQueue, ctrl.pvQueue)

	return ctrl
}

// Run starts CSI attacher and listens on channel events
func (ctrl *CSIAttachController) Run(ctx context.Context, workers int, wg *sync.WaitGroup) {
	defer ctrl.vaQueue.ShutDown()
	defer ctrl.pvQueue.ShutDown()

	logger := klog.FromContext(ctx)
	logger.Info("Starting CSI attacher")
	defer logger.Info("Shutting CSI attacher")

	if !cache.WaitForCacheSync(ctx.Done(), ctrl.vaListerSynced, ctrl.pvListerSynced) {
		logger.Error(nil, "Cannot sync caches")
		return
	}
	if utilfeature.DefaultFeatureGate.Enabled(features.ReleaseLeaderElectionOnExit) {
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				wait.UntilWithContext(ctx, ctrl.syncVA, 0)
			}()
			wg.Add(1)
			go func() {
				defer wg.Done()
				wait.UntilWithContext(ctx, ctrl.syncPV, 0)
			}()
		}

		if ctrl.shouldReconcileVolumeAttachment {
			wg.Add(1)
			go func() {
				defer wg.Done()
				wait.UntilWithContext(ctx, func(ctx context.Context) {
					err := ctrl.handler.ReconcileVA(ctx)
					if err != nil {
						logger.Error(err, "Failed to reconcile VolumeAttachment")
					}
				}, ctrl.reconcileSync)
			}()
		}
	} else {
		for range workers {
			go wait.UntilWithContext(ctx, ctrl.syncVA, 0)
			go wait.UntilWithContext(ctx, ctrl.syncPV, 0)
		}

		if ctrl.shouldReconcileVolumeAttachment {
			go wait.UntilWithContext(ctx, func(ctx context.Context) {
				err := ctrl.handler.ReconcileVA(ctx)
				if err != nil {
					logger.Error(err, "Failed to reconcile VolumeAttachment")
				}
			}, ctrl.reconcileSync)
		}
	}

	<-ctx.Done()
}

// vaAdded reacts to a VolumeAttachment creation
func (ctrl *CSIAttachController) vaAdded(obj any) {
	va := obj.(*storage.VolumeAttachment)
	ctrl.vaQueue.Add(va.Name)
}

// vaUpdated return a function that reacts to a VolumeAttachment update
func (ctrl *CSIAttachController) vaUpdatedFunc(logger klog.Logger) func(old, new any) {
	return func(old, new any) {
		oldVA := old.(*storage.VolumeAttachment)
		newVA := new.(*storage.VolumeAttachment)
		if shouldEnqueueVAChange(oldVA, newVA) {
			ctrl.vaQueue.Add(newVA.Name)
		} else {
			logger.V(3).Info("Ignoring VolumeAttachment change", "VolumeAttachment", newVA.Name)
		}
	}
}

// vaDeleted reacts to a VolumeAttachment deleted
func (ctrl *CSIAttachController) vaDeleted(obj any) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}
	va := obj.(*storage.VolumeAttachment)
	if va != nil && va.Spec.Source.PersistentVolumeName != nil {
		// Enqueue PV sync event - it will evaluate and remove finalizer
		ctrl.pvQueue.Add(*va.Spec.Source.PersistentVolumeName)
	}
}

// pvAdded reacts to a PV creation
func (ctrl *CSIAttachController) pvAdded(obj any) {
	pv := obj.(*v1.PersistentVolume)
	if !ctrl.processFinalizers(pv) {
		return
	}
	ctrl.pvQueue.Add(pv.Name)
}

// pvUpdated reacts to a PV update
func (ctrl *CSIAttachController) pvUpdated(old, new any) {
	pv := new.(*v1.PersistentVolume)
	if !ctrl.processFinalizers(pv) {
		return
	}
	ctrl.pvQueue.Add(pv.Name)
}

// syncVA deals with one key off the queue.  It returns false when it's time to quit.
func (ctrl *CSIAttachController) syncVA(ctx context.Context) {
	vaName, quit := ctrl.vaQueue.Get()
	if quit {
		return
	}
	defer ctrl.vaQueue.Done(vaName)

	logger := klog.LoggerWithValues(klog.FromContext(ctx), "VolumeAttachment", vaName)
	ctx = klog.NewContext(ctx, logger)
	logger.V(4).Info("Started VolumeAttachment processing")

	// get VolumeAttachment to process
	va, err := ctrl.vaLister.Get(vaName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			// VolumeAttachment was deleted in the meantime, ignore.
			logger.V(3).Info("VolumeAttachment was deleted, ignoring")
			return
		}
		logger.Error(err, "Error getting VolumeAttachment")
		ctrl.vaQueue.AddRateLimited(vaName)
		return
	}
	if va.Spec.Attacher != ctrl.attacherName {
		logger.V(4).Info("Skipping VolumeAttachment for attacher", "attacher", va.Spec.Attacher)
		return
	}
	ctrl.handler.SyncNewOrUpdatedVolumeAttachment(ctx, va)
}

func (ctrl *CSIAttachController) processFinalizers(pv *v1.PersistentVolume) bool {
	if sets.NewString(pv.Finalizers...).Has(GetFinalizerName(ctrl.attacherName)) {
		if pv.DeletionTimestamp != nil {
			return true
		}

		// if PV is provisioned by in-tree plugin and does not have migrated-to label
		// this normally means this is a rollback scenario, we need to remove the finalizer as well
		if ctrl.translator.IsPVMigratable(pv) {
			if ann := pv.Annotations; ann != nil {
				if migratedToDriver := ann[annMigratedTo]; migratedToDriver == ctrl.attacherName {
					// migrated-to annonation detected, keep the finalizer
					return false
				}
			}
			return true
		}
	}
	return false
}

// syncPV deals with one key off the queue.  It returns false when it's time to quit.
func (ctrl *CSIAttachController) syncPV(ctx context.Context) {
	pvName, quit := ctrl.pvQueue.Get()
	if quit {
		return
	}
	defer ctrl.pvQueue.Done(pvName)

	logger := klog.LoggerWithValues(klog.FromContext(ctx), "PersistentVolume", pvName)
	ctx = klog.NewContext(ctx, logger)
	logger.V(4).Info("Started PersistentVolume processing")

	// get PV to process
	pv, err := ctrl.pvLister.Get(pvName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			// PV was deleted in the meantime, ignore.
			logger.V(3).Info("PersistentVolume was deleted, ignoring")
			return
		}
		logger.Error(err, "Error getting PersistentVolume")
		ctrl.pvQueue.AddRateLimited(pvName)
		return
	}
	ctrl.handler.SyncNewOrUpdatedPersistentVolume(ctx, pv)
}

// shouldEnqueueVAChange checks if a changed VolumeAttachment should be enqueued.
// It filters out changes in Status.Attach/DetachError - these were posted by the controller
// just few moments ago. If they were enqueued, Attach()/Detach() would be called again,
// breaking exponential backoff.
func shouldEnqueueVAChange(old, new *storage.VolumeAttachment) bool {
	if old.ResourceVersion == new.ResourceVersion {
		// This is most probably periodic sync, enqueue it
		return true
	}
	if new.Status.AttachError == nil && new.Status.DetachError == nil && old.Status.AttachError == nil && old.Status.DetachError == nil {
		// The difference between old and new must be elsewhere than Status.Attach/DetachError
		return true
	}

	sanitized := new.DeepCopy()
	sanitized.ResourceVersion = old.ResourceVersion
	sanitized.Status.AttachError = old.Status.AttachError
	sanitized.Status.DetachError = old.Status.DetachError
	sanitized.ManagedFields = old.ManagedFields

	if equality.Semantic.DeepEqual(old, sanitized) {
		// The objects are the same except Status.Attach/DetachError.
		// Don't enqueue them.
		return false
	}
	return true
}
