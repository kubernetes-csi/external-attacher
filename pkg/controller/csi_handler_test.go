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
	"fmt"
	"testing"
	"time"

	"github.com/kubernetes-csi/external-attacher/pkg/attacher"

	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/testing"
	"k8s.io/klog"
)

const (
	// Finalizer value
	fin = "external-attacher/csi-test"
)

var (
	ann = map[string]string{
		vaNodeIDAnnotation: "nodeID1",
	}
)

var timeout = 10 * time.Millisecond

func csiHandlerFactory(client kubernetes.Interface, informerFactory informers.SharedInformerFactory, csi attacher.Attacher) Handler {
	return NewCSIHandler(
		client,
		testAttacherName,
		csi,
		informerFactory.Core().V1().PersistentVolumes().Lister(),
		informerFactory.Core().V1().Nodes().Lister(),
		informerFactory.Storage().V1beta1().CSINodes().Lister(),
		informerFactory.Storage().V1beta1().VolumeAttachments().Lister(),
		&timeout,
		true, /* supports PUBLISH_READONLY */
	)
}

func csiHandlerFactoryNoReadOnly(client kubernetes.Interface, informerFactory informers.SharedInformerFactory, csi attacher.Attacher) Handler {
	return NewCSIHandler(
		client,
		testAttacherName,
		csi,
		informerFactory.Core().V1().PersistentVolumes().Lister(),
		informerFactory.Core().V1().Nodes().Lister(),
		informerFactory.Storage().V1beta1().CSINodes().Lister(),
		informerFactory.Storage().V1beta1().VolumeAttachments().Lister(),
		&timeout,
		false, /* does not support PUBLISH_READONLY */
	)
}

func pv() *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: testPVName,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       testAttacherName,
					VolumeHandle: testVolumeHandle,
					ReadOnly:     false,
				},
			},
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteMany,
			},
		},
	}
}

func gcePDPV() *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: testPVName,
			Labels: map[string]string{
				"failure-domain.beta.kubernetes.io/zone": "testZone",
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				GCEPersistentDisk: &v1.GCEPersistentDiskVolumeSource{
					PDName:    "testpd",
					FSType:    "ext4",
					Partition: 0,
					ReadOnly:  false,
				},
			},
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
		},
	}
}

func gcePDPVWithFinalizer() *v1.PersistentVolume {
	pv := gcePDPV()
	pv.Finalizers = []string{fin}
	return pv
}

func pvWithFinalizer() *v1.PersistentVolume {
	pv := pv()
	pv.Finalizers = []string{fin}
	return pv
}

func pvWithFinalizers(pv *v1.PersistentVolume, finalizers ...string) *v1.PersistentVolume {
	pv.Finalizers = append(pv.Finalizers, finalizers...)
	return pv
}

func pvDeleted(pv *v1.PersistentVolume) *v1.PersistentVolume {
	pv.DeletionTimestamp = &metav1.Time{}
	return pv
}

func pvWithAttributes(pv *v1.PersistentVolume, attributes map[string]string) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource.CSI.VolumeAttributes = attributes
	return pv
}

func vaInlineSpecWithAttributes(va *storage.VolumeAttachment, attributes map[string]string) *storage.VolumeAttachment {
	va.Spec.Source.InlineVolumeSpec.PersistentVolumeSource.CSI.VolumeAttributes = attributes
	return va
}

func pvWithSecret(pv *v1.PersistentVolume, secretName string) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource.CSI.ControllerPublishSecretRef = &v1.SecretReference{
		Name:      secretName,
		Namespace: "default",
	}
	return pv
}

func vaInlineSpecWithSecret(va *storage.VolumeAttachment, secretName string) *storage.VolumeAttachment {
	va.Spec.Source.InlineVolumeSpec.PersistentVolumeSource.CSI.ControllerPublishSecretRef = &v1.SecretReference{
		Name:      secretName,
		Namespace: "default",
	}
	return va
}

func pvReadOnly(pv *v1.PersistentVolume) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource.CSI.ReadOnly = true
	return pv
}

func vaInlineSpecReadOnly(va *storage.VolumeAttachment) *storage.VolumeAttachment {
	va.Spec.Source.InlineVolumeSpec.PersistentVolumeSource.CSI.ReadOnly = true
	return va
}

func node() *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
			Annotations: map[string]string{
				"csi.volume.kubernetes.io/nodeid": fmt.Sprintf("{ %q: %q }", testAttacherName, testNodeID),
			},
		},
	}
}

func nodeWithoutAnnotations() *v1.Node {
	n := node()
	n.Annotations = nil
	return n
}

func csiNode() *storage.CSINode {
	return &storage.CSINode{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
		Spec: storage.CSINodeSpec{
			Drivers: []storage.CSINodeDriver{
				{
					Name:   testAttacherName,
					NodeID: testNodeID,
				},
			},
		},
	}
}

func csiNodeEmpty() *storage.CSINode {
	return &storage.CSINode{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
		Spec: storage.CSINodeSpec{Drivers: []storage.CSINodeDriver{}},
	}
}

func emptySecret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emptySecret",
			Namespace: "default",
		},
	}
}

func secret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"foo": []byte("bar"),
		},
	}
}

func patch(original, new interface{}) []byte {
	patch, err := createMergePatch(original, new)
	if err != nil {
		klog.Fatalf("Failed to create patch %+v", err)
		return nil
	}
	return patch
}

func TestCSIHandler(t *testing.T) {
	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1beta1",
		Resource: "volumeattachments",
	}
	pvGroupResourceVersion := schema.GroupVersionResource{
		Group:    v1.GroupName,
		Version:  "v1",
		Resource: "persistentvolumes",
	}
	secretGroupResourceVersion := schema.GroupVersionResource{
		Group:    v1.GroupName,
		Version:  "v1",
		Resource: "secrets",
	}

	var noMetadata map[string]string
	var noAttrs map[string]string
	var noSecrets map[string]string
	var notDetached = false
	var detached = true
	var success error
	var readWrite = false
	var readOnly = true
	var ignored = false // the value is irrelevant for given call

	tests := []testCase{
		//
		// ATTACH
		//
		{
			name:           "VolumeAttachment added -> successful attachment",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec -> successful attachment",
			initialObjects: []runtime.Object{node()},
			addedVA:        vaWithInlineSpec(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "readOnly VolumeAttachment added -> successful attachment",
			initialObjects: []runtime.Object{pvReadOnly(pvWithFinalizer()), node()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readOnly, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "readOnly VolumeAttachment with InlineVolumeSpec -> successful attachment",
			initialObjects: []runtime.Object{node()},
			addedVA:        vaInlineSpecReadOnly(vaWithInlineSpec(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */))),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readOnly, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment updated -> successful attachment",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec updated -> successful attachment",
			initialObjects: []runtime.Object{node()},
			updatedVA:      vaWithInlineSpec(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */)),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with attributes -> successful attachment",
			initialObjects: []runtime.Object{pvWithAttributes(pvWithFinalizer(), map[string]string{"foo": "bar"}), node()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, map[string]string{"foo": "bar"}, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec and attributes -> successful attachment",
			initialObjects: []runtime.Object{node()},
			updatedVA:      vaInlineSpecWithAttributes(vaWithInlineSpec(va(false, "", nil)) /*va*/, map[string]string{"foo": "bar"} /*attributes*/),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, map[string]string{"foo": "bar"}, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with secrets -> successful attachment",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "secret"), node(), secret()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec and secrets -> successful attachment",
			initialObjects: []runtime.Object{node(), secret()},
			updatedVA:      vaInlineSpecWithSecret(vaWithInlineSpec(va(false, "", nil)) /*va*/, "secret" /*secret*/),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with empty secrets -> successful attachment",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "emptySecret"), node(), emptySecret()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec and empty secrets -> successful attachment",
			initialObjects: []runtime.Object{node(), emptySecret()},
			updatedVA:      vaInlineSpecWithSecret(vaWithInlineSpec(va(false, "", nil)) /*va*/, "emptySecret" /*secret*/),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with missing secrets -> error",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "unknownSecret"), node()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "unknownSecret"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "", nil),
						vaWithAttachError(va(false, "", nil), "failed to load secret \"default/unknownSecret\": secrets \"unknownSecret\" not found"))),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "VolumeAttachment updated -> PV finalizer is added",
			initialObjects: []runtime.Object{pv(), node()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				// PV Finalizer after VA
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pv(), pvWithFinalizer())),
				// VA Finalizer is saved last
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "error saving PV finalizer -> controller retries",
			initialObjects: []runtime.Object{pv(), node()},
			updatedVA:      va(false, "", nil),
			reactors: []reaction{
				{
					verb:     "patch",
					resource: "persistentvolumes",
					reactor: func(t *testing.T) core.ReactionFunc {
						i := 0
						return func(core.Action) (bool, runtime.Object, error) {
							i++
							if i < 2 {
								// Update fails once
								return true, nil, apierrors.NewForbidden(v1.Resource("persistentvolume"), "pv1", errors.New("Mock error"))
							}
							// Update succeeds for the 2nd time
							return false, nil, nil
						}
					},
				},
			},
			expectedActions: []core.Action{
				// PV Finalizer - fails
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pv(), pvWithFinalizer())),
				// Error is saved
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						vaWithAttachError(va(false, "", nil), "could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock error"))),
				// Second PV Finalizer - succeeds
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pv(), pvWithFinalizer())),
				// VA Finalizer is saved first, error remains
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						vaWithAttachError(va(false, "", nil),
							"could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock error"),
						vaWithAttachError(va(false, fin, ann),
							"could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock error"))),
				// Attach succeeds, error is deleted
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						vaWithAttachError(va(false, fin, ann),
							"could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock error"),
						va(true, fin, ann)),
				),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:             "already attached volume -> ignored",
			initialObjects:   []runtime.Object{pvWithFinalizer(), node()},
			updatedVA:        va(true, fin, ann),
			expectedActions:  []core.Action{},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "PV with deletion timestamp -> ignored with error",
			initialObjects: []runtime.Object{pvDeleted(pv()), node()},
			updatedVA:      va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						vaWithAttachError(va(false, fin, ann), "PersistentVolume \"pv1\" is marked for deletion"))),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "VolumeAttachment added -> successful attachment incl. metadata",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "", nil), va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						vaWithMetadata(va(true, fin, ann), map[string]string{"foo": "bar"}))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, map[string]string{"foo": "bar"}, 0},
			},
		},
		{
			name:            "unknown driver -> ignored",
			initialObjects:  []runtime.Object{pvWithFinalizer(), node()},
			addedVA:         vaWithInvalidDriver(va(false, fin, ann)),
			expectedActions: []core.Action{},
		},
		{
			name:           "unknown PV -> error",
			initialObjects: []runtime.Object{node()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"persistentvolume \"pv1\" not found"))),
			},
		},
		{
			name:           "unknown PV -> error + error saving the error",
			initialObjects: []runtime.Object{node()},
			addedVA:        va(false, fin, ann),
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
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"persistentvolume \"pv1\" not found"))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"persistentvolume \"pv1\" not found"))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"persistentvolume \"pv1\" not found"))),
			},
		},
		{
			name:           "neither PV nor InlineVolumeSpec reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)),
						vaWithAttachError(vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)), "neither InlineCSIVolumeSource nor PersistentVolumeName specified in VA source"))),
			},
		},
		{
			name:           "both PV and InlineVolumeSpec reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        vaAddInlineSpec(va(false, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)),
						vaWithAttachError(vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)), "both InlineCSIVolumeSource and PersistentVolumeName specified in VA source"))),
			},
		},
		{
			name:           "unknown node -> error",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"node \"node1\" not found"))),
			},
		},
		{
			name:           "failed write with VA finializers -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false, "", nil),
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
				// Controller tries to save VA finalizer, it fails
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, "", nil),
						va(false /*attached*/, fin, ann))),
				// Controller tries to save error, it fails too

				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						vaWithAttachError(va(false /*attached*/, "", nil), "could not save VolumeAttachment: volumeattachments.storage.k8s.io \"pv1-node1\" is forbidden: Mock error"))),
				// Controller tries to save VA finalizer again, it succeeds
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "failed write with attached=true -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false, "", nil),
			reactors: []reaction{
				{
					verb:     "patch",
					resource: "volumeattachments",
					reactor: func(t *testing.T) core.ReactionFunc {
						i := 0
						return func(core.Action) (bool, runtime.Object, error) {
							i++
							if i != 2 {
								return false, nil, nil
							}
							return true, nil, apierrors.NewForbidden(storage.Resource("volumeattachments"), "pv1-node1", errors.New("mock error"))
						}
					},
				},
			},
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				// Second save with attached=true fails
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						va(true /*attached*/, fin, ann))),
				// Our implementation of fake PATCH did not store the first VA with annotation + finalizer,
				// the controller tries to save it again.
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				// Final save that succeeds.
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "CSI attach fails -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						vaWithAttachError(va(false, fin, ann), "mock error"))),
				// Our implementation of fake PATCH did not store the first VA with annotation + finalizer,
				// the controller tries to save it again.
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				// Final save that succeeds.
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(vaWithAttachError(va(false, fin, ann), "mock error"),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, fmt.Errorf("mock error"), notDetached, noMetadata, 0},
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "CSI attach times out -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 500 * time.Millisecond},
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, time.Duration(0)},
			},
		},
		{
			name:           "Node without annotations -> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), nodeWithoutAnnotations()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						vaWithAttachError(va(false, fin, ann), "node \"node1\" has no NodeID annotation"))),
			},
		},
		{
			name:           "CSINode exists without the driver, Node without annotations -> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), nodeWithoutAnnotations(), csiNodeEmpty()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						vaWithAttachError(va(false, fin, ann), "node \"node1\" has no NodeID annotation"))),
			},
		},
		{
			name:           "CSINode exists with the driver, Node without annotations -> success",
			initialObjects: []runtime.Object{pvWithFinalizer(), nodeWithoutAnnotations(), csiNode()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with GCEPersistentDiskVolumeSource -> successful attachment",
			initialObjects: []runtime.Object{gcePDPVWithFinalizer(), node()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", "projects/UNSPECIFIED/zones/testZone/disks/testpd", testNodeID,
					map[string]string{"partition": ""}, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},

		//DETACH

		{
			name:           "VolumeAttachment marked for deletion -> successful detach",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(false /*attached*/, "", ann)))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec marked for deletion -> successful detach",
			initialObjects: []runtime.Object{node()},
			addedVA:        deleted(vaWithInlineSpec(va(true, fin, ann))),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaWithInlineSpec(va(true, fin, ann))),
						deleted(vaWithInlineSpec(va(false /*attached*/, "", ann))))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume with secrets -> successful detach",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "secret"), node(), secret()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(false /*attached*/, "", ann)))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume attachment with InlineVolumeSpec and secrets -> successful detach",
			initialObjects: []runtime.Object{node(), secret()},
			addedVA:        deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, fin, ann)), "secret")),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, fin, ann)), "secret")),
						deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(false /*attached*/, "", ann)), "secret")))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume with empty secrets -> successful detach",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "emptySecret"), node(), emptySecret()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(false /*attached*/, "", ann)))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume attachment with InlineVolumeSpec and empty secrets -> successful detach",
			initialObjects: []runtime.Object{node(), emptySecret()},
			addedVA:        deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, fin, ann)), "emptySecret")),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, fin, ann)), "emptySecret")),
						deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(false /*attached*/, "", ann)), "emptySecret")))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume with missing secrets -> error",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "unknownSecret"), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "unknownSecret"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann),
							"failed to load secret \"default/unknownSecret\": secrets \"unknownSecret\" not found")))),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "CSI detach fails with an error -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "mock error")))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(false, "", ann)))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, fmt.Errorf("mock error"), ignored, noMetadata, 0},
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "CSI detach times out -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(false /*attached*/, "", ann)))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, 500 * time.Millisecond},
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, time.Duration(0)},
			},
		},
		{
			name:             "already detached volume -> ignored",
			initialObjects:   []runtime.Object{pvWithFinalizer(), node()},
			updatedVA:        deleted(va(false, "", nil)),
			expectedActions:  []core.Action{},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "detach unknown PV -> error",
			initialObjects: []runtime.Object{node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found")))),
			},
		},
		{
			name:           "detach unknown PV -> error + error saving the error",
			initialObjects: []runtime.Object{node()},
			addedVA:        deleted(va(true, fin, ann)),
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
								return true, nil, apierrors.NewForbidden(storage.Resource("volumeattachments"), "pv1-node1", errors.New("Mock error"))
							}
							// Update succeeds for the 3rd time
							return false, nil, nil
						}
					},
				},
			},
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found")))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found")))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found")))),
			},
		},
		{
			name:           "detach VA with neither PV nor InlineCSIVolumeSource reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(vaWithNoPVReferenceNorInlineVolumeSpec(va(true, fin, ann))),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaWithNoPVReferenceNorInlineVolumeSpec(va(true, fin, ann))),
						deleted(vaWithDetachError(vaWithNoPVReferenceNorInlineVolumeSpec(va(true, fin, ann)),
							"neither InlineCSIVolumeSource nor PersistentVolumeName specified in VA source")))),
			},
		},
		{
			name:           "detach VA with both PV and InlineCSIVolumeSource reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(vaAddInlineSpec(va(true, fin, ann))),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaAddInlineSpec(va(true, fin, ann))),
						deleted(vaWithDetachError(vaAddInlineSpec(va(true, fin, ann)),
							"both InlineCSIVolumeSource and PersistentVolumeName specified in VA source")))),
			},
		},
		{
			name:           "detach unknown node -> error",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        deleted(va(true, fin, nil)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, nil)),
						deleted(vaWithDetachError(va(true, fin, nil), "node \"node1\" not found")))),
			},
		},
		{
			name:           "detach unknown node -> use annotation",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        deleted(va(true, fin, map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, fin, map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})),
						deleted(va(false /*attached*/, "", map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, "annotatedNodeID", noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment marked for deletion -> node is preferred over VA annotation for NodeID",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(true, fin, map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})),
						deleted(va(false /*attached*/, "", map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "failed write with attached=false -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(false, fin, ann)),
			reactors: []reaction{
				{
					verb:     "patch",
					resource: "volumeattachments",
					reactor: func(t *testing.T) core.ReactionFunc {
						i := 0
						return func(core.Action) (bool, runtime.Object, error) {
							i++
							if i == 1 {
								return true, nil, apierrors.NewForbidden(storage.Resource("volumeattachments"), "pv1-node1", errors.New("mock error"))
							}
							return false, nil, nil
						}
					},
				},
			},
			expectedActions: []core.Action{
				// This fails
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(false, fin, ann)),
						deleted(va(false, "", ann)))),
				// Saving error succeeds
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(false, fin, ann)),
						vaWithDetachError(deleted(va(false, fin, ann)),
							"could not mark as detached: volumeattachments.storage.k8s.io \"pv1-node1\" is forbidden: mock error"))),
				// Second save of attached=false succeeds
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(false, fin, ann)),
						deleted(va(false, "", ann)))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with GCEPersistentDiskVolumeSource marked for deletion -> successful detach",
			initialObjects: []runtime.Object{gcePDPVWithFinalizer(), node()},
			addedVA:        deleted(va(true /*attached*/, fin /*finalizer*/, ann)),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, fin, ann)),
						deleted(va(false /*attached*/, "", ann)))),
			},
			expectedCSICalls: []csiCall{
				{"detach", "projects/UNSPECIFIED/zones/testZone/disks/testpd", testNodeID,
					map[string]string{"partition": "0"}, noSecrets, readWrite, success, detached, noMetadata, 0},
			},
		},
		//
		// PV finalizers
		//
		{
			name:           "VA deleted -> PV finalizer removed",
			initialObjects: []runtime.Object{pvDeleted(pvWithFinalizer())},
			deletedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pvDeleted(pvWithFinalizer()),
						pvDeleted(pv()))),
			},
		},
		{
			name:           "PV updated -> PV finalizer removed",
			initialObjects: []runtime.Object{},
			updatedPV:      pvDeleted(pvWithFinalizer()),
			expectedActions: []core.Action{
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pvDeleted(pvWithFinalizer()),
						pvDeleted(pv()))),
			},
		},
		{
			name:           "PV finalizer removed -> other finalizers preserved",
			initialObjects: []runtime.Object{pvDeleted(pvWithFinalizers(pvWithFinalizer(), "foo/bar", "bar/baz"))},
			deletedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pvDeleted(pvWithFinalizers(pvWithFinalizer(), "foo/bar", "bar/baz")), pvDeleted(pvWithFinalizers(pv(), "foo/bar", "bar/baz")))),
			},
		},
		{
			name:           "finalizer removal fails -> controller retries",
			initialObjects: []runtime.Object{pvDeleted(pvWithFinalizer())},
			deletedVA:      va(false, "", nil),
			reactors: []reaction{
				{
					verb:     "patch",
					resource: "persistentvolumes",
					reactor: func(t *testing.T) core.ReactionFunc {
						i := 0
						return func(core.Action) (bool, runtime.Object, error) {
							i++
							if i < 3 {
								return true, nil, apierrors.NewForbidden(v1.Resource("persistentvolumes"), "pv1", errors.New("mock error"))
							}
							return false, nil, nil
						}
					},
				},
			},
			expectedActions: []core.Action{
				// This update fails
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pvDeleted(pvWithFinalizer()), pvDeleted(pv()))),
				// This one fails too
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pvDeleted(pvWithFinalizer()), pvDeleted(pv()))),
				// This one succeeds
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pvDeleted(pvWithFinalizer()), pvDeleted(pv()))),
			},
		},
		{
			name:            "no PV finalizer -> ignored",
			initialObjects:  []runtime.Object{pvDeleted(pv())},
			deletedVA:       va(false, "", nil),
			expectedActions: []core.Action{},
		},
		{
			name:            "no deletion timestamp -> ignored",
			initialObjects:  []runtime.Object{pv()},
			deletedVA:       va(false, "", nil),
			expectedActions: []core.Action{},
		},
		{
			name:            "VA exists -> ignored",
			initialObjects:  []runtime.Object{pvDeleted(pvWithFinalizer()), va(false, "", nil)},
			deletedVA:       va(false, "", nil),
			expectedActions: []core.Action{},
		},
	}

	runTests(t, csiHandlerFactory, tests)
}

func TestCSIHandlerReadOnly(t *testing.T) {
	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1beta1",
		Resource: "volumeattachments",
	}
	var noMetadata map[string]string
	var noAttrs map[string]string
	var noSecrets map[string]string
	var notDetached = false
	var success error
	var readWrite = false

	tests := []testCase{
		//
		// ATTACH with driver that does not support PUBLISH_READONLY
		//
		{
			name:           "read-write PV -> attached as read-write",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "read-only PV -> attached as read-write",
			initialObjects: []runtime.Object{pvReadOnly(pvWithFinalizer()), node()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann))),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
	}
	runTests(t, csiHandlerFactoryNoReadOnly, tests)
}
