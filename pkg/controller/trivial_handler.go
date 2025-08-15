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
	vaQueue, pvQueue workqueue.TypedRateLimitingInterface[string]
}

var _ Handler = &trivialHandler{}

// NewTrivialHandler provides new Handler for Volumeattachments and PV object handling.
func NewTrivialHandler(client kubernetes.Interface) Handler {
	return &trivialHandler{client: client}
}

func (h *trivialHandler) Init(vaQueue workqueue.TypedRateLimitingInterface[string], pvQueue workqueue.TypedRateLimitingInterface[string]) {
	h.vaQueue = vaQueue
	h.pvQueue = pvQueue
}

func (h *trivialHandler) ReconcileVA(ctx context.Context) error {
	return nil
}

func (h *trivialHandler) SyncNewOrUpdatedVolumeAttachment(ctx context.Context, va *storage.VolumeAttachment) {
	logger := klog.FromContext(ctx)
	ctx = klog.NewContext(ctx, logger)
	logger.V(4).Info("Trivial sync started")
	if !va.Status.Attached {
		// mark as attached
		if _, err := markAsAttached(ctx, h.client, va, nil); err != nil {
			logger.Error(err, "Error saving VolumeAttachment as attached")
			h.vaQueue.AddRateLimited(va.Name)
			return
		}
		logger.V(2).Info("Marked VolumeAttachment as attached")
	}
	h.vaQueue.Forget(va.Name)
}

func (h *trivialHandler) SyncNewOrUpdatedPersistentVolume(ctx context.Context, pv *v1.PersistentVolume) {
}
