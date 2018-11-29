/*
Copyright 2018 The Kubernetes Authors.

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
	"testing"

	storage "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldEnqueueVAChange(t *testing.T) {
	va1 := &storage.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "foo",
			ResourceVersion: "1",
		},
		Spec: storage.VolumeAttachmentSpec{
			Attacher: "1",
		},
		Status: storage.VolumeAttachmentStatus{
			Attached: false,
		},
	}

	va1WithAttachError := va1.DeepCopy()
	va1.Status.AttachError = &storage.VolumeError{
		Message: "mock error1",
		Time:    metav1.Time{},
	}

	va1WithDetachError := va1.DeepCopy()
	va1.Status.DetachError = &storage.VolumeError{
		Message: "mock error1",
		Time:    metav1.Time{},
	}

	va2ChangedSpec := va1.DeepCopy()
	va2ChangedSpec.ResourceVersion = "2"
	va2ChangedSpec.Spec.Attacher = "2"

	va2ChangedMetadata := va1.DeepCopy()
	va2ChangedMetadata.ResourceVersion = "2"
	va2ChangedMetadata.Annotations = map[string]string{"foo": "bar"}

	va2ChangedAttachError := va1.DeepCopy()
	va2ChangedAttachError.ResourceVersion = "2"
	va2ChangedAttachError.Status.AttachError = &storage.VolumeError{
		Message: "mock error2",
		Time:    metav1.Time{},
	}

	va2ChangedDetachError := va1.DeepCopy()
	va2ChangedDetachError.ResourceVersion = "2"
	va2ChangedDetachError.Status.DetachError = &storage.VolumeError{
		Message: "mock error2",
		Time:    metav1.Time{},
	}

	tests := []struct {
		name           string
		oldVA, newVA   *storage.VolumeAttachment
		expectedResult bool
	}{
		{
			name:           "periodic sync",
			oldVA:          va1,
			newVA:          va1,
			expectedResult: true,
		},
		{
			name:           "changed spec",
			oldVA:          va1,
			newVA:          va2ChangedSpec,
			expectedResult: true,
		},
		{
			name:           "changed metadata",
			oldVA:          va1,
			newVA:          va2ChangedMetadata,
			expectedResult: true,
		},
		{
			name:           "added attachError",
			oldVA:          va1,
			newVA:          va2ChangedAttachError,
			expectedResult: false,
		},
		{
			name:           "added detachError",
			oldVA:          va1,
			newVA:          va2ChangedDetachError,
			expectedResult: false,
		},
		{
			name:           "changed attachError",
			oldVA:          va1WithAttachError,
			newVA:          va2ChangedAttachError,
			expectedResult: false,
		},
		{
			name:           "changed detachError",
			oldVA:          va1WithDetachError,
			newVA:          va2ChangedDetachError,
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := shouldEnqueueVAChange(test.oldVA, test.newVA)
			if result != test.expectedResult {
				t.Errorf("Error: expected result %t, got %t", test.expectedResult, result)
			}
		})
	}
}
