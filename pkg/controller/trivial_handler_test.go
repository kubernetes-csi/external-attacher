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
	"errors"
	"testing"

	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	core "k8s.io/client-go/testing"
)

const (
	testAttacherName = "csi/test"
	testPVName       = "pv1"
	testNodeName     = "node1"
)

func createVolumeAttachment(attacher string, pvName string, nodeName string, attached bool) *storagev1.VolumeAttachment {
	return &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName + "-" + nodeName,
		},
		Spec: storagev1.VolumeAttachmentSpec{
			Attacher: attacher,
			NodeName: nodeName,
			AttachedVolumeSource: storagev1.AttachedVolumeSource{
				PersistentVolumeName: &pvName,
			},
		},
		Status: storagev1.VolumeAttachmentStatus{
			Attached: attached,
		},
	}
}

func va(attached bool) *storagev1.VolumeAttachment {
	return createVolumeAttachment(testAttacherName, testPVName, testNodeName, attached)
}

func invalidDriverVA() *storagev1.VolumeAttachment {
	return createVolumeAttachment("unknownDriver", testPVName, testNodeName, false)
}

func TestTrivialHandler(t *testing.T) {
	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storagev1.GroupName,
		Version:  "v1",
		Resource: "volumeattachments",
	}

	tests := []testCase{
		{
			name:    "add -> successful write",
			addedVa: va(false),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true)),
			},
		},
		{
			name:      "update -> successful write",
			updatedVa: va(false),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true)),
			},
		},
		{
			name:            "unknown driver -> controller ignores",
			addedVa:         invalidDriverVA(),
			expectedActions: []core.Action{},
		},
		{
			name:    "failed write -> controller retries",
			addedVa: va(false),
			reactors: []reaction{
				{
					verb:     "update",
					resource: "volumeattachments",
					reactor: func(t *testing.T) core.ReactionFunc {
						i := 0
						return func(core.Action) (bool, runtime.Object, error) {
							i++
							if i < 3 {
								// Update fails 2 times
								return true, nil, apierrors.NewForbidden(storagev1.Resource("volumeattachments"), "pv1-node1", errors.New("Mock error"))
							}
							// Update succeeds for the 3rd time
							return false, nil, nil
						}
					},
				},
			},
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true)),
			},
		},
	}

	runTests(t, tests)
}
