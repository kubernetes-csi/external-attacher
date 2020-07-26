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
	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// trivialHandler is a handler that marks all VolumeAttachments as attached.
// It's used for CSI drivers that don't support ControllerPublishVolume call.
// It uses no finalizer, deletion of VolumeAttachment is instant (as there is
// nothing to detach).
type trivialHandler struct {
	client           kubernetes.Interface
	vaQueue, pvQueue workqueue.RateLimitingInterface
}

var _ Handler = &trivialHandler{}

// NewTrivialHandler provides new Handler for Volumeattachments and PV object handling.
func NewTrivialHandler(client kubernetes.Interface) Handler {
	return &trivialHandler{client: client}
}

func (h *trivialHandler) Init(vaQueue workqueue.RateLimitingInterface, pvQueue workqueue.RateLimitingInterface) {
	h.vaQueue = vaQueue
	h.pvQueue = pvQueue
}

func (h *trivialHandler) ReconcileVA() error {
	return nil
}

func (h *trivialHandler) SyncNewOrUpdatedVolumeAttachment(va *storage.VolumeAttachment) {
	klog.V(4).Infof("Trivial sync[%s] started", va.Name)
	if !va.Status.Attached {
		// mark as attached
		if _, err := markAsAttached(h.client, va, nil); err != nil {
			klog.Warningf("Error saving VolumeAttachment %s as attached: %s", va.Name, err)
			h.vaQueue.AddRateLimited(va.Name)
			return
		}
		klog.V(2).Infof("Marked VolumeAttachment %s as attached", va.Name)
	}
	h.vaQueue.Forget(va.Name)
}

func (h *trivialHandler) SyncNewOrUpdatedPersistentVolume(pv *v1.PersistentVolume) {
	return
}
