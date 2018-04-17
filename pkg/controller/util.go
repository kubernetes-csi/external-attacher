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
	"github.com/golang/glog"
	"github.com/kubernetes-csi/external-attacher/pkg/connection"
	storage "k8s.io/api/storage/v1beta1"
	"k8s.io/client-go/kubernetes"
)

func markAsAttached(client kubernetes.Interface, va *storage.VolumeAttachment, metadata map[string]string) (*storage.VolumeAttachment, error) {
	glog.V(4).Infof("Marking as attached %q", va.Name)
	clone := va.DeepCopy()
	clone.Status.Attached = true
	clone.Status.AttachmentMetadata = metadata
	clone.Status.AttachError = nil
	// TODO: use patch to save us from VersionError
	newVA, err := client.StorageV1beta1().VolumeAttachments().Update(clone)
	if err != nil {
		return va, err
	}
	glog.V(4).Infof("Marked as attached %q", va.Name)
	return newVA, nil
}

func markAsDetached(client kubernetes.Interface, va *storage.VolumeAttachment) (*storage.VolumeAttachment, error) {
	finalizerName := connection.GetFinalizerName(va.Spec.Attacher)

	// Prepare new array of finalizers
	newFinalizers := make([]string, 0, len(va.Finalizers))
	found := false
	for _, f := range va.Finalizers {
		if f == finalizerName {
			found = true
			continue
		}
		newFinalizers = append(newFinalizers, f)
	}
	// Mostly to simplify unit tests, but it won't harm in production too
	if len(newFinalizers) == 0 {
		newFinalizers = nil
	}

	if !found && !va.Status.Attached {
		// Finalizer was not present, nothing to update
		glog.V(4).Infof("Already fully detached %q", va.Name)
		return va, nil
	}

	glog.V(4).Infof("Marking as detached %q", va.Name)
	clone := va.DeepCopy()
	clone.Finalizers = newFinalizers
	clone.Status.Attached = false
	clone.Status.DetachError = nil
	clone.Status.AttachmentMetadata = nil
	// TODO: use patch to save us from VersionError
	newVA, err := client.StorageV1beta1().VolumeAttachments().Update(clone)
	if err != nil {
		return va, err
	}
	glog.V(4).Infof("Finalizer removed from %q", va.Name)
	return newVA, nil
}
