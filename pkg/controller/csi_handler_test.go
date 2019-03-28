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

	"k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/testing"
	csiapi "k8s.io/csi-api/pkg/apis/csi/v1alpha1"
	csiclient "k8s.io/csi-api/pkg/client/clientset/versioned"
	csiinformers "k8s.io/csi-api/pkg/client/informers/externalversions"
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

func csiHandlerFactory(client kubernetes.Interface, csiClient csiclient.Interface, informerFactory informers.SharedInformerFactory, csiInformerFactory csiinformers.SharedInformerFactory, csi attacher.Attacher) Handler {
	return NewCSIHandler(
		client,
		csiClient,
		testAttacherName,
		csi,
		informerFactory.Core().V1().PersistentVolumes().Lister(),
		informerFactory.Core().V1().Nodes().Lister(),
		csiInformerFactory.Csi().V1alpha1().CSINodeInfos().Lister(),
		informerFactory.Storage().V1beta1().VolumeAttachments().Lister(),
		&timeout,
		true, /* supports PUBLISH_READONLY */
	)
}

func csiHandlerFactoryNoReadOnly(client kubernetes.Interface, csiClient csiclient.Interface, informerFactory informers.SharedInformerFactory, csiInformerFactory csiinformers.SharedInformerFactory, csi attacher.Attacher) Handler {
	return NewCSIHandler(
		client,
		csiClient,
		testAttacherName,
		csi,
		informerFactory.Core().V1().PersistentVolumes().Lister(),
		informerFactory.Core().V1().Nodes().Lister(),
		csiInformerFactory.Csi().V1alpha1().CSINodeInfos().Lister(),
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

func pvWithSecret(pv *v1.PersistentVolume, secretName string) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource.CSI.ControllerPublishSecretRef = &v1.SecretReference{
		Name:      secretName,
		Namespace: "default",
	}
	return pv
}

func pvReadOnly(pv *v1.PersistentVolume) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource.CSI.ReadOnly = true
	return pv
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

func csiNodeInfo() *csiapi.CSINodeInfo {
	return &csiapi.CSINodeInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
		Spec: csiapi.CSINodeInfoSpec{
			Drivers: []csiapi.CSIDriverInfoSpec{
				{
					Name:   testAttacherName,
					NodeID: testNodeID,
				},
			},
		},
	}
}

func csiNodeInfoEmpty() *csiapi.CSINodeInfo {
	return &csiapi.CSINodeInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNodeName,
		},
		Spec: csiapi.CSINodeInfoSpec{Drivers: []csiapi.CSIDriverInfoSpec{}},
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, "", nil), "failed to load secret \"default/unknownSecret\": secrets \"unknownSecret\" not found")),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "VolumeAttachment updated -> PV finalizer is added",
			initialObjects: []runtime.Object{pv(), node()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				// PV Finalizer after VA
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvWithFinalizer()),
				// VA Finalizer is saved last
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
					verb:     "update",
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
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvWithFinalizer()),
				// Error is saved
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, "", nil), "could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock error")),
				// Second PV Finalizer - succeeds
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvWithFinalizer()),
				// VA Finalizer is saved first, error remains
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock error")),
				// Attach succeeds, error is deleted
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "PersistentVolume \"pv1\" is marked for deletion")),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "VolumeAttachment added -> successful attachment incl. metadata",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithMetadata(va(true, fin, ann), map[string]string{"foo": "bar"})),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "persistentvolume \"pv1\" not found")),
			},
		},
		{
			name:           "unknown PV -> error + error saving the error",
			initialObjects: []runtime.Object{node()},
			addedVA:        va(false, fin, ann),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "persistentvolume \"pv1\" not found")),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "persistentvolume \"pv1\" not found")),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "persistentvolume \"pv1\" not found")),
			},
		},
		{
			name:           "invalid PV reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        vaWithNoPVReference(va(false, fin, ann)),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(vaWithNoPVReference(va(false, fin, ann)), "VolumeAttachment.spec.persistentVolumeName is empty")),
			},
		},
		{
			name:           "unknown node -> error",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "node \"node1\" not found")),
			},
		},
		{
			name:           "failed write with VA finializers -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false, "", nil),
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
				// Controller tries to save VA finalizer, it fails
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				// Controller tries to save error, it fails too
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false /*attached*/, "", nil), "could not save VolumeAttachment: volumeattachments.storage.k8s.io \"pv1-node1\" is forbidden: Mock error")),
				// Controller tries to save VA finalizer again, it succeeds
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
					verb:     "update",
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				// Second save with attached=true fails
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "mock error")),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "node \"node1\" has no NodeID annotation")),
			},
		},
		{
			name:           "CSINodeInfo exists without the driver, Node without annotations -> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), nodeWithoutAnnotations(), csiNodeInfoEmpty()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithAttachError(va(false, fin, ann), "node \"node1\" has no NodeID annotation")),
			},
		},
		{
			name:           "CSINodeInfo exists with the driver, Node without annotations -> success",
			initialObjects: []runtime.Object{pvWithFinalizer(), nodeWithoutAnnotations(), csiNodeInfo()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
			},
			expectedCSICalls: []csiCall{
				{"attach", "projects/UNSPECIFIED/zones/testZone/disks/testpd", testNodeID,
					map[string]string{"partition": ""}, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		//
		// DETACH
		//
		{
			name:           "VolumeAttachment marked for deletion -> successful detach",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", ann))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "volume with secrets -> successful detach",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "secret"), node(), secret()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", ann))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "volume with empty secrets -> successful detach",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "emptySecret"), node(), emptySecret()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", ann))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "volume with missing secrets -> error",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "unknownSecret"), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "unknownSecret"),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(vaWithDetachError(va(true, fin, ann), "failed to load secret \"default/unknownSecret\": secrets \"unknownSecret\" not found"))),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "CSI detach fails with transient error -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithDetachError(deleted(va(true /*attached*/, fin, ann)), "mock error")),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", ann))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, fmt.Errorf("mock error"), notDetached, noMetadata, 0},
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "CSI detach times out -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", ann))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, 500 * time.Millisecond},
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, time.Duration(0)},
			},
		},
		{
			name:           "CSI detach fails with final error -> controller does not retry",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", ann))),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, fmt.Errorf("mock error"), detached, noMetadata, 0},
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found"))),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found"))),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found"))),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found"))),
			},
		},
		{
			name:           "detach invalid PV reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        deleted(vaWithNoPVReference(va(true, fin, ann))),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(vaWithDetachError(vaWithNoPVReference(va(true, fin, ann)), "VolumeAttachment.spec.persistentVolumeName is empty"))),
			},
		},
		{
			name:           "detach unknown node -> error",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        deleted(va(true, fin, nil)),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(vaWithDetachError(va(true, fin, nil), "node \"node1\" not found"))),
			},
		},
		{
			name:           "detach unknown node -> use annotation",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        deleted(va(true, fin, map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})),
			expectedActions: []core.Action{
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", map[string]string{vaNodeIDAnnotation: "annotatedNodeID"}))),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", map[string]string{vaNodeIDAnnotation: "annotatedNodeID"}))),
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
					verb:     "update",
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false, "", ann))),
				// Saving error succeeds
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, vaWithDetachError(deleted(va(false, fin, ann)), "could not mark as detached: volumeattachments.storage.k8s.io \"pv1-node1\" is forbidden: mock error")),
				// Second save of attached=false succeeds
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false, "", ann))),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, deleted(va(false /*attached*/, "", ann))),
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
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvDeleted(pv())),
			},
		},
		{
			name:           "PV updated -> PV finalizer removed",
			initialObjects: []runtime.Object{},
			updatedPV:      pvDeleted(pvWithFinalizer()),
			expectedActions: []core.Action{
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvDeleted(pv())),
			},
		},
		{
			name:           "PV finalizer removed -> other finalizers preserved",
			initialObjects: []runtime.Object{pvDeleted(pvWithFinalizers(pvWithFinalizer(), "foo/bar", "bar/baz"))},
			deletedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvDeleted(pvWithFinalizers(pv(), "foo/bar", "bar/baz"))),
			},
		},
		{
			name:           "finalizer removal fails -> controller retries",
			initialObjects: []runtime.Object{pvDeleted(pvWithFinalizer())},
			deletedVA:      va(false, "", nil),
			reactors: []reaction{
				{
					verb:     "update",
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
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvDeleted(pv())),
				// This one fails too
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvDeleted(pv())),
				// This one succeeds
				core.NewUpdateAction(pvGroupResourceVersion, metav1.NamespaceNone, pvDeleted(pv())),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
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
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(false /*attached*/, fin, ann)),
				core.NewUpdateAction(vaGroupResourceVersion, metav1.NamespaceNone, va(true /*attached*/, fin, ann)),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
	}
	runTests(t, csiHandlerFactoryNoReadOnly, tests)
}
