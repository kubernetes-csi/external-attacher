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

	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	csitrans "k8s.io/csi-translation-lib"
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

	va2AppendManagedFields := va1.DeepCopy()
	va2AppendManagedFields.ResourceVersion = "2"
	va2AppendManagedFields.ManagedFields = append(va2AppendManagedFields.ManagedFields,
		metav1.ManagedFieldsEntry{
			APIVersion: "storage.k8s.io/v1beta1",
			Manager:    "csi-attacher",
			Operation:  "Update",
			FieldsType: "FieldsV1",
			FieldsV1: &metav1.FieldsV1{
				Raw: []byte(`{"f:metadata":{"f:annotations":{".":{},"f:csi.alpha.kubernetes.io/node-id":{}},"f:finalizers":{".":{},"v:\\\"external-attacher/csi-cdsplugin\\\"":{}}},"f:status":{"f:attached":{},"f:attachmentMetadata":{".":{},"f:devName":{},"f:serial":{}}}}`),
			},
		})

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
		{
			name:           "appended managedFields",
			oldVA:          va1,
			newVA:          va2AppendManagedFields,
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

func TestProcessFinalizers(t *testing.T) {
	type testcase struct {
		name           string
		pv             *v1.PersistentVolume
		expectedResult bool
	}

	c := &CSIAttachController{}
	c.translator = csitrans.New()
	c.attacherName = "pd.csi.storage.gke.io"
	time := metav1.Now()

	testcases := []testcase{
		{
			name:           "nothing interesting in the PV",
			pv:             &v1.PersistentVolume{},
			expectedResult: false,
		},
		{
			name: "no deletion timestamp, has finalizer",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"external-attacher/pd-csi-storage-gke-io"},
				},
			},
			expectedResult: false,
		},
		{
			name: "Has deletion timestamp, has finalizer",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &time,
					Finalizers:        []string{"external-attacher/pd-csi-storage-gke-io"},
				},
			},
			expectedResult: true,
		},
		{
			name: "no deletion timestamp, has finalizer, migrated PV, no migrated-to annotation",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"external-attacher/pd-csi-storage-gke-io"},
				},
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						GCEPersistentDisk: &v1.GCEPersistentDiskVolumeSource{},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "no deletion timestamp, has finalizer, migrated PV, no migrated-to annotation with random anno",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers:  []string{"external-attacher/pd-csi-storage-gke-io"},
					Annotations: map[string]string{"random": "random"},
				},
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						GCEPersistentDisk: &v1.GCEPersistentDiskVolumeSource{},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "no deletion timestamp, has finalizer, migrated PV, has migrated-to annotation",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"external-attacher/pd-csi-storage-gke-io"},
					Annotations: map[string]string{
						"pv.kubernetes.io/migrated-to": "pd.csi.storage.gke.io",
					},
				},
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						GCEPersistentDisk: &v1.GCEPersistentDiskVolumeSource{},
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, tc := range testcases {
		result := c.processFinalizers(tc.pv)
		if result != tc.expectedResult {
			t.Errorf("Error executing test %v: expected result %v, got %v", tc.name, tc.expectedResult, result)
		}
	}
}
