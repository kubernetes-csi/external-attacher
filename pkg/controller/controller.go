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
	"fmt"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	storageinformers "k8s.io/client-go/informers/storage/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type CSIAttachController struct {
	client        kubernetes.Interface
	attacherName  string
	handler       Handler
	eventRecorder record.EventRecorder
	vaQueue       workqueue.RateLimitingInterface
	pvQueue       workqueue.RateLimitingInterface

	vaLister       storagelisters.VolumeAttachmentLister
	vaListerSynced cache.InformerSynced
	pvLister       corelisters.PersistentVolumeLister
	pvListerSynced cache.InformerSynced
}

// Handler is responsible for handling VolumeAttachment events from informer.
type Handler interface {
	Init(vaQueue workqueue.RateLimitingInterface, pvQueue workqueue.RateLimitingInterface)

	// SyncNewOrUpdatedVolumeAttachment processes one Add/Updated event from
	// VolumeAttachment informers. It runs in a workqueue, guaranting that only
	// one SyncNewOrUpdatedVolumeAttachment runs for given VA.
	// SyncNewOrUpdatedVolumeAttachment is responsible for marking the
	// VolumeAttachment either as forgotten (resets exponential backoff) or
	// re-queue it into the vaQueue to process it after exponential
	// backoff.
	SyncNewOrUpdatedVolumeAttachment(va *storage.VolumeAttachment)

	SyncNewOrUpdatedPersistentVolume(pv *v1.PersistentVolume)
}

// NewCSIAttachController returns a new *CSIAttachController
func NewCSIAttachController(client kubernetes.Interface, attacherName string, handler Handler, volumeAttachmentInformer storageinformers.VolumeAttachmentInformer, pvInformer coreinformers.PersistentVolumeInformer) *CSIAttachController {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: client.Core().Events(v1.NamespaceAll)})
	var eventRecorder record.EventRecorder
	eventRecorder = broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: fmt.Sprintf("csi-attacher %s", attacherName)})

	ctrl := &CSIAttachController{
		client:        client,
		attacherName:  attacherName,
		handler:       handler,
		eventRecorder: eventRecorder,
		vaQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "csi-attacher-va"),
		pvQueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "csi-attacher-pv"),
	}

	volumeAttachmentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.vaAdded,
		UpdateFunc: ctrl.vaUpdated,
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

func (ctrl *CSIAttachController) Run(workers int, stopCh <-chan struct{}) {
	defer ctrl.vaQueue.ShutDown()
	defer ctrl.pvQueue.ShutDown()

	glog.Infof("Starting CSI attacher")
	defer glog.Infof("Shutting CSI attacher")

	if !cache.WaitForCacheSync(stopCh, ctrl.vaListerSynced, ctrl.pvListerSynced) {
		glog.Errorf("Cannot sync caches")
		return
	}
	for i := 0; i < workers; i++ {
		go wait.Until(ctrl.syncVA, 0, stopCh)
		go wait.Until(ctrl.syncPV, 0, stopCh)
	}

	<-stopCh
}

// vaAdded reacts to a VolumeAttachment creation
func (ctrl *CSIAttachController) vaAdded(obj interface{}) {
	va := obj.(*storage.VolumeAttachment)
	ctrl.vaQueue.Add(va.Name)
}

// vaUpdated reacts to a VolumeAttachment update
func (ctrl *CSIAttachController) vaUpdated(old, new interface{}) {
	va := new.(*storage.VolumeAttachment)
	ctrl.vaQueue.Add(va.Name)
}

// vaDeleted reacts to a VolumeAttachment deleted
func (ctrl *CSIAttachController) vaDeleted(obj interface{}) {
	va := obj.(*storage.VolumeAttachment)
	if va != nil && va.Spec.Source.PersistentVolumeName != nil {
		// Enqueue PV sync event - it will evaluate and remove finalizer
		ctrl.pvQueue.Add(*va.Spec.Source.PersistentVolumeName)
	}
}

// pvAdded reacts to a PV creation
func (ctrl *CSIAttachController) pvAdded(obj interface{}) {
	pv := obj.(*v1.PersistentVolume)
	ctrl.pvQueue.Add(pv.Name)
}

// pvUpdated reacts to a PV update
func (ctrl *CSIAttachController) pvUpdated(old, new interface{}) {
	pv := new.(*v1.PersistentVolume)
	ctrl.pvQueue.Add(pv.Name)
}

// syncVA deals with one key off the queue.  It returns false when it's time to quit.
func (ctrl *CSIAttachController) syncVA() {
	key, quit := ctrl.vaQueue.Get()
	if quit {
		return
	}
	defer ctrl.vaQueue.Done(key)

	vaName := key.(string)
	glog.V(4).Infof("Started VA processing %q", vaName)

	// get VolumeAttachment to process
	va, err := ctrl.vaLister.Get(vaName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			// VolumeAttachment was deleted in the meantime, ignore.
			glog.V(3).Infof("VA %q deleted, ignoring", vaName)
			return
		}
		glog.Errorf("Error getting VolumeAttachment %q: %v", vaName, err)
		ctrl.vaQueue.AddRateLimited(vaName)
		return
	}
	if va.Spec.Attacher != ctrl.attacherName {
		glog.V(4).Infof("Skipping VolumeAttachment %s for attacher %s", va.Name, va.Spec.Attacher)
		return
	}
	ctrl.handler.SyncNewOrUpdatedVolumeAttachment(va)
}

// syncPV deals with one key off the queue.  It returns false when it's time to quit.
func (ctrl *CSIAttachController) syncPV() {
	key, quit := ctrl.pvQueue.Get()
	if quit {
		return
	}
	defer ctrl.pvQueue.Done(key)

	pvName := key.(string)
	glog.V(4).Infof("Started PV processing %q", pvName)

	// get PV to process
	pv, err := ctrl.pvLister.Get(pvName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			// PV was deleted in the meantime, ignore.
			glog.V(3).Infof("PV %q deleted, ignoring", pvName)
			return
		}
		glog.Errorf("Error getting PersistentVolume %q: %v", pvName, err)
		ctrl.pvQueue.AddRateLimited(pvName)
		return
	}
	ctrl.handler.SyncNewOrUpdatedPersistentVolume(pv)
}
