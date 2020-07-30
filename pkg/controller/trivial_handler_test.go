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

	"github.com/kubernetes-csi/external-attacher/pkg/attacher"

	storage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/testing"
)

func trivialHandlerFactory(client kubernetes.Interface, informerFactory informers.SharedInformerFactory, csi attacher.Attacher, lister VolumeLister) Handler {
	return NewTrivialHandler(client)
}

func TestTrivialHandler(t *testing.T) {
	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1",
		Resource: "volumeattachments",
	}

	tests := []testCase{
		{
			name:    "add -> successful write",
			addedVA: va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						va(false, "", nil),
						va(true, "", nil)), "status"),
			},
		},
		{
			name:      "update -> successful write",
			updatedVA: va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						va(false, "", nil),
						va(true, "", nil)), "status"),
			},
		},
		{
			name:            "unknown driver -> controller ignores",
			addedVA:         vaWithInvalidDriver(va(false, "", nil)),
			expectedActions: []core.Action{},
		},
		{
			name:    "failed write -> controller retries",
			addedVA: va(false, "", nil),
			reactors: []reaction{
				{
					verb:     "patch",
					resource: "volumeattachments",
					reactor: func(t *testing.T) core.ReactionFunc {
						i := 0
						return func(core.Action) (bool, runtime.Object, error) {
							i++
							if i < 3 {
								// Update fails 2 times
								return true, nil, apierrors.NewForbidden(storage.Resource("volumeattachments"), "pv1-node1", errors.New("Mock error"))
							}
							// Update succeeds for the 3rd time
							return false, nil, nil
						}
					},
				},
			},
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						va(false, "", nil),
						va(true, "", nil)), "status"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						va(false, "", nil),
						va(true, "", nil)), "status"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						va(false, "", nil),
						va(true, "", nil)), "status"),
			},
		},
	}

	runTests(t, trivialHandlerFactory, tests)
}
