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
	"time"

	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	storageinformerv1 "k8s.io/client-go/informers/storage/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	storagelisterv1 "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type CSIAttachController struct {
	client        kubernetes.Interface
	attacherName  string
	handler       Handler
	eventRecorder record.EventRecorder
	queue         workqueue.RateLimitingInterface
	syncFunc      func(va *storagev1.VolumeAttachment, queue workqueue.RateLimitingInterface) (finished bool, err error)

	vaLister       storagelisterv1.VolumeAttachmentLister
	vaListerSynced cache.InformerSynced
}

// Handler is responsible for handling VolumeAttachment events from informer.
type Handler interface {
	// SyncNewOrUpdatedVolumeAttachment processes one Add/Updated event from
	// VolumeAttachment informers. It runs in a workqueue, guaranting that only
	// one SyncNewOrUpdatedVolumeAttachment runs for given VA.
	// SyncNewOrUpdatedVolumeAttachment is responsible for marking the
	// VolumeAttachment either as forgotten (resets exponential backoff) or
	// re-queue it into the provided queue to process it after exponential
	// backoff.
	SyncNewOrUpdatedVolumeAttachment(va *storagev1.VolumeAttachment, queue workqueue.RateLimitingInterface)
}

// NewCSIAttachController returns a new *CSIAttachController
func NewCSIAttachController(client kubernetes.Interface, attacherName string, handler Handler, volumeAttachmentInformer storageinformerv1.VolumeAttachmentInformer) *CSIAttachController {
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: client.Core().Events(v1.NamespaceAll)})
	var eventRecorder record.EventRecorder
	eventRecorder = broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: fmt.Sprintf("csi-attacher %s", attacherName)})

	ctrl := &CSIAttachController{
		client:        client,
		attacherName:  attacherName,
		handler:       handler,
		eventRecorder: eventRecorder,
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "csi-attacher"),
	}

	volumeAttachmentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.vaAdded,
		UpdateFunc: ctrl.vaUpdated,
		// DeleteFunc: ctrl.vaDeleted, // TODO: do we need this?
	})
	ctrl.vaLister = volumeAttachmentInformer.Lister()
	ctrl.vaListerSynced = volumeAttachmentInformer.Informer().HasSynced

	return ctrl
}

func (ctrl *CSIAttachController) Run(workers int, stopCh <-chan struct{}) {
	defer ctrl.queue.ShutDown()

	glog.Infof("Starting CSI attacher")
	defer glog.Infof("Shutting CSI attacher")

	if !cache.WaitForCacheSync(stopCh, ctrl.vaListerSynced) {
		glog.Errorf("Cannot sync caches")
		return
	}
	for i := 0; i < workers; i++ {
		go wait.Until(ctrl.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

// vaAdded reacts to a VolumeAttachment creation
func (ctrl *CSIAttachController) vaAdded(obj interface{}) {
	va := obj.(*storagev1.VolumeAttachment)
	ctrl.queue.Add(va.Name)
}

// vaUpdated reacts to a VolumeAttachment update
func (ctrl *CSIAttachController) vaUpdated(old, new interface{}) {
	va := new.(*storagev1.VolumeAttachment)
	ctrl.queue.Add(va.Name)
}

func (ctrl *CSIAttachController) runWorker() {
	for ctrl.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (ctrl *CSIAttachController) processNextWorkItem() bool {
	key, quit := ctrl.queue.Get()
	if quit {
		return false
	}
	defer ctrl.queue.Done(key)

	vaName := key.(string)
	glog.V(4).Infof("Started processing %q", vaName)

	// get VolumeAttachment to process
	va, err := ctrl.vaLister.Get(vaName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			// VolumeAttachment was deleted in the meantime, ignore.
			glog.V(3).Infof("%q deleted, ignoring", vaName)
			return true
		}
		glog.Errorf("Error getting VolumeAttachment %q: %v", vaName, err)
		ctrl.queue.AddRateLimited(vaName)
		return true
	}
	if va.Spec.Attacher != ctrl.attacherName {
		glog.V(4).Infof("Skipping VolumeAttachment %s for attacher %s", va.Name, va.Spec.Attacher)
		return true
	}
	ctrl.handler.SyncNewOrUpdatedVolumeAttachment(va, ctrl.queue)
	return true
}
