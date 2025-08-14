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
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/external-attacher/pkg/attacher"
	"github.com/kubernetes-csi/external-attacher/pkg/features"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	core "k8s.io/client-go/testing"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	csitranslator "k8s.io/csi-translation-lib"
	"k8s.io/klog/v2"
	_ "k8s.io/klog/v2/ktesting/init"
)

const (
	// Finalizer value
	fin = "external-attacher/csi-test"

	defaultFSType = "ext4"
)

var (
	ann = map[string]string{
		vaNodeIDAnnotation: "nodeID1",
	}
)

var timeout = 10 * time.Millisecond

func csiHandlerFactory(client kubernetes.Interface, informerFactory informers.SharedInformerFactory, csi attacher.Attacher, lister VolumeLister) Handler {
	return NewCSIHandler(
		client,
		testAttacherName,
		csi,
		lister,
		informerFactory.Core().V1().PersistentVolumes().Lister(),
		informerFactory.Storage().V1().CSINodes().Lister(),
		informerFactory.Storage().V1().VolumeAttachments().Lister(),
		&timeout,
		true,  /* supports PUBLISH_READONLY */
		false, /* does not support SINGLE_NODE_MULTI_WRITER access mode */
		csitranslator.New(),
		defaultFSType,
	)
}

func csiHandlerFactoryNoReadOnly(client kubernetes.Interface, informerFactory informers.SharedInformerFactory, csi attacher.Attacher, lister VolumeLister) Handler {
	return NewCSIHandler(
		client,
		testAttacherName,
		csi,
		lister,
		informerFactory.Core().V1().PersistentVolumes().Lister(),
		informerFactory.Storage().V1().CSINodes().Lister(),
		informerFactory.Storage().V1().VolumeAttachments().Lister(),
		&timeout,
		false, /* does not support PUBLISH_READONLY */
		false, /* does not support SINGLE_NODE_MULTI_WRITER access mode */
		csitranslator.New(),
		defaultFSType,
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

func pvWithDriverName(driver string) *v1.PersistentVolume {
	pv := pv()
	pv.Spec.CSI.Driver = driver
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
		},
	}
}

func nodeWithAnnotations() *v1.Node {
	node := node()
	node.Annotations = map[string]string{
		"csi.volume.kubernetes.io/nodeid": fmt.Sprintf("{ %q: %q }", testAttacherName, testNodeID),
	}
	return node
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

func patch(original, new any) []byte {
	patch, err := createMergePatch(original, new)
	if err != nil {
		klog.Background().Error(err, "Failed to create patch")
		return nil
	}
	return patch
}

func vaWithAttachErrorAndCode(va *storage.VolumeAttachment, message string, code codes.Code) *storage.VolumeAttachment {
	errorCode := int32(code)
	va.Status.AttachError = &storage.VolumeError{
		Message:   message,
		Time:      metav1.Time{},
		ErrorCode: &errorCode,
	}
	return va
}

func TestCSIHandler(t *testing.T) {
	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1",
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
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec -> successful attachment",
			initialObjects: []runtime.Object{csiNode()},
			addedVA:        vaWithInlineSpec(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "readOnly VolumeAttachment added -> successful attachment",
			initialObjects: []runtime.Object{pvReadOnly(pvWithFinalizer()), csiNode()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readOnly, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "readOnly VolumeAttachment with InlineVolumeSpec -> successful attachment",
			initialObjects: []runtime.Object{csiNode()},
			addedVA:        vaInlineSpecReadOnly(vaWithInlineSpec(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */))),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readOnly, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment updated -> successful attachment",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec updated -> successful attachment",
			initialObjects: []runtime.Object{csiNode()},
			updatedVA:      vaWithInlineSpec(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */)),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with attributes -> successful attachment",
			initialObjects: []runtime.Object{pvWithAttributes(pvWithFinalizer(), map[string]string{"foo": "bar"}), csiNode()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, map[string]string{"foo": "bar"}, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec and attributes -> successful attachment",
			initialObjects: []runtime.Object{csiNode()},
			updatedVA:      vaInlineSpecWithAttributes(vaWithInlineSpec(va(false, "", nil)) /*va*/, map[string]string{"foo": "bar"} /*attributes*/),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, map[string]string{"foo": "bar"}, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with secrets -> successful attachment",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "secret"), secret(), csiNode()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec and secrets -> successful attachment",
			initialObjects: []runtime.Object{secret(), csiNode()},
			updatedVA:      vaInlineSpecWithSecret(vaWithInlineSpec(va(false, "", nil)) /*va*/, "secret" /*secret*/),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with empty secrets -> successful attachment",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "emptySecret"), emptySecret(), csiNode()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec and empty secrets -> successful attachment",
			initialObjects: []runtime.Object{emptySecret(), csiNode()},
			updatedVA:      vaInlineSpecWithSecret(vaWithInlineSpec(va(false, "", nil)) /*va*/, "emptySecret" /*secret*/),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with missing secrets -> error",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "unknownSecret"), csiNode()},
			updatedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "unknownSecret"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "", nil),
						vaWithAttachError(va(false, "", nil),
							"failed to load secret \"default/unknownSecret\": secrets \"unknownSecret\" not found")),
					"status"),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "VolumeAttachment updated -> PV finalizer is added",
			initialObjects: []runtime.Object{pv(), csiNode()},
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
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "error saving PV finalizer -> controller retries",
			initialObjects: []runtime.Object{pv(), csiNode()},
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
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						vaWithAttachError(va(false, "", nil),
							"could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock"+
								" error")), "status"),
				// Second PV Finalizer - succeeds
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pv(), pvWithFinalizer())),
				// VA Finalizer is saved first, error remains
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						vaWithAttachError(va(false, "", nil),
							"could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock error"),
						vaWithAttachError(va(false, fin, ann),
							"could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock"+
								" error"))),
				// Attach succeeds, error is deleted
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						vaWithAttachError(va(false, fin, ann),
							"could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock error"),
						va(true, fin, ann)), "status")},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:             "already attached volume -> ignored",
			initialObjects:   []runtime.Object{pvWithFinalizer(), csiNode()},
			updatedVA:        va(true, fin, ann),
			expectedActions:  []core.Action{},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "PV with deletion timestamp -> ignored with error",
			initialObjects: []runtime.Object{pvDeleted(pv()), csiNode()},
			updatedVA:      va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						vaWithAttachError(va(false, fin, ann), "PersistentVolume \"pv1\" is marked for deletion")),
					"status"),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "VolumeAttachment added -> successful attachment incl. metadata",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "", nil), va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						vaWithMetadata(va(true, fin, ann), map[string]string{"foo": "bar"})), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, map[string]string{"foo": "bar"}, 0},
			},
		},
		{
			name:            "unknown driver -> ignored",
			initialObjects:  []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:         vaWithInvalidDriver(va(false, fin, ann)),
			expectedActions: []core.Action{},
		},
		{
			name:           "unknown PV -> error",
			initialObjects: []runtime.Object{csiNode()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"persistentvolume \"pv1\" not found")), "status"),
			},
		},
		{
			name:           "unknown PV -> error + error saving the error",
			initialObjects: []runtime.Object{csiNode()},
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
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"persistentvolume \"pv1\" not found")), "status"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"persistentvolume \"pv1\" not found")), "status"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"persistentvolume \"pv1\" not found")), "status"),
			},
		},
		{
			name:           "neither PV nor InlineVolumeSpec reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)),
						vaWithAttachError(vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)),
							"neither InlineCSIVolumeSource nor PersistentVolumeName specified in VA source")), "status"),
			},
		},
		{
			name:           "both PV and InlineVolumeSpec reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        vaAddInlineSpec(va(false, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)),
						vaWithAttachError(vaWithNoPVReferenceNorInlineVolumeSpec(va(false, fin, ann)),
							"both InlineCSIVolumeSource and PersistentVolumeName specified in VA source")), "status"),
			},
		},
		{
			name:           "unknown node -> error",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), vaWithAttachError(va(false, fin, ann),
						"csinode.storage.k8s.io \"node1\" not found")), "status"),
			},
		},
		{
			name:           "failed write with VA finializers -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
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

				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						vaWithAttachError(va(false /*attached*/, "", nil),
							"could not save VolumeAttachment: volumeattachments.storage.k8s."+
								"io \"pv1-node1\" is forbidden: Mock error")), "status"),
				// Controller tries to save VA finalizer again, it succeeds
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "failed write with attached=true -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
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
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						va(true /*attached*/, fin, ann)), "status"),
				// Our implementation of fake PATCH did not store the first VA with annotation + finalizer,
				// the controller tries to save it again.
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				// Final save that succeeds.
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "CSI attach fails -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						vaWithAttachError(va(false, fin, ann), "mock error")),
					"status"),
				// Our implementation of fake PATCH did not store the first VA with annotation + finalizer,
				// the controller tries to save it again.
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				// Final save that succeeds.
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(vaWithAttachError(va(false, fin, ann), "mock error"),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, fmt.Errorf("mock error"), notDetached, noMetadata, 0},
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "CSI attach times out -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				// Add finalizer again (see: https://github.com/kubernetes-csi/external-attacher/issues/228)
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin /*finalizer*/, ann /* annotations */),
						vaWithAttachError(va(false, fin, ann), "context deadline exceeded")),
					"status"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(vaWithAttachError(va(false, fin, ann), "context deadline exceeded"),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 500 * time.Millisecond},
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, time.Duration(0)},
			},
		},
		{
			name:           "Node without CSINode -> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), node()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						vaWithAttachError(va(false, fin, ann), "csinode.storage.k8s.io \"node1\" not found")), "status"),
			},
		},
		{
			name:           "Node with annotations, CSINode is absent -> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), nodeWithAnnotations()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						vaWithAttachError(va(false, fin, ann), "csinode.storage.k8s.io \"node1\" not found")),
					"status"),
			},
		},
		{
			name:           "CSINode exists without the driver -> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNodeEmpty()},
			addedVA:        va(false, fin, ann),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						vaWithAttachError(va(false, fin, ann), "CSINode node1 does not contain driver csi/test")),
					"status"),
			},
		},
		{
			name:           "CSINode exists with the driver, Node without annotations -> success",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with GCEPersistentDiskVolumeSource -> successful attachment",
			initialObjects: []runtime.Object{gcePDPVWithFinalizer(), csiNode()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", "projects/UNSPECIFIED/zones/testZone/disks/testpd", testNodeID,
					map[string]string{"partition": ""}, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},

		//DETACH

		{
			name:           "VolumeAttachment marked for deletion -> successful detach",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(true /*attached*/, "", ann)))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, "", ann)),
						deleted(va(false /*attached*/, "", ann))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with InlineVolumeSpec marked for deletion -> successful detach",
			initialObjects: []runtime.Object{csiNode()},
			addedVA:        deleted(vaWithInlineSpec(va(true, fin, ann))),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaWithInlineSpec(va(true, fin, ann))),
						deleted(vaWithInlineSpec(va(true /*attached*/, "", ann))))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaWithInlineSpec(va(true, "", ann))),
						deleted(vaWithInlineSpec(va(false /*attached*/, "", ann)))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume with secrets -> successful detach",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "secret"), secret()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaWithInlineSpec(va(true, fin, ann))),
						deleted(vaWithInlineSpec(va(true /*attached*/, "", ann))))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, "", ann)),
						deleted(va(false /*attached*/, "", ann))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume attachment with InlineVolumeSpec and secrets -> successful detach",
			initialObjects: []runtime.Object{secret()},
			addedVA:        deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, fin, ann)), "secret")),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "secret"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, fin, ann)), "secret")),
						deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true /*attached*/, "", ann)),
							"secret"))), ""),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, "", ann)), "secret")),
						deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(false /*attached*/, "", ann)),
							"secret"))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{"foo": "bar"}, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume with empty secrets -> successful detach",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "emptySecret"), emptySecret()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(true /*attached*/, "", ann)))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, "", ann)),
						deleted(va(false /*attached*/, "", ann))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume attachment with InlineVolumeSpec and empty secrets -> successful detach",
			initialObjects: []runtime.Object{emptySecret()},
			addedVA:        deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, fin, ann)), "emptySecret")),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "emptySecret"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, fin, ann)), "emptySecret")),
						deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true /*attached*/, "", ann)),
							"emptySecret"))), ""),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(true, "", ann)), "emptySecret")),
						deleted(vaInlineSpecWithSecret(vaWithInlineSpec(va(false /*attached*/, "", ann)),
							"emptySecret"))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, map[string]string{}, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "volume with missing secrets -> error",
			initialObjects: []runtime.Object{pvWithSecret(pvWithFinalizer(), "unknownSecret"), csiNode()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewGetAction(secretGroupResourceVersion, "default", "unknownSecret"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann),
							"failed to load secret \"default/unknownSecret\": secrets \"unknownSecret\" not found"+
								""))), "status"),
			},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "CSI detach fails with an error -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, "", ann)),
						deleted(vaWithDetachError(va(true, "", ann), "mock error"))), "status"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(true, "", ann)))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaWithDetachError(va(true, "", ann), "mock error")),
						deleted(va(false, "", ann))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, fmt.Errorf("mock error"), ignored, noMetadata, 0},
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, 0},
			},
		},
		{
			name:           "CSI detach times out -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, "", ann)),
						deleted(vaWithDetachError(va(true, "", ann), "context deadline exceeded"))), "status"),
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(va(true /*attached*/, "", ann)))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaWithDetachError(va(true, "", ann), "context deadline exceeded")),
						deleted(va(false /*attached*/, "", ann))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, 500 * time.Millisecond},
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, ignored, noMetadata, time.Duration(0)},
			},
		},
		{
			name:             "already detached volume -> ignored",
			initialObjects:   []runtime.Object{pvWithFinalizer(), csiNode()},
			updatedVA:        deleted(va(false, "", nil)),
			expectedActions:  []core.Action{},
			expectedCSICalls: []csiCall{},
		},
		{
			name:           "detach unknown PV -> error",
			initialObjects: []runtime.Object{csiNode()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, "", ann)),
						deleted(vaWithDetachError(va(true, "", ann), "persistentvolume \"pv1\" not found"))),
					"status"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(false, "", ann)),
						deleted(vaWithDetachError(va(false, "", ann), "persistentvolume \"pv1\" not found"))),
					"status"),
			},
		},
		{
			name:           "detach unknown PV -> error + error saving the error",
			initialObjects: []runtime.Object{csiNode()},
			addedVA:        deleted(va(true, fin, ann)),
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
			// The handler will perform the same patch regardless of whether the error save was successful or not. The only
			// difference is in the errors logged (which are not checked here).
			// Because the detach never succeeds, the test will loop as long as there are expected actions remaining.
			// 4 such loops are tested below: two when the error save fails, and then two when the error succeeds.
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found"))),
					"status"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found"))),
					"status"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found"))),
					"status"),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)),
						deleted(vaWithDetachError(va(true, fin, ann), "persistentvolume \"pv1\" not found"))),
					"status"),
			},
		},
		{
			name:           "detach VA with neither PV nor InlineCSIVolumeSource reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        deleted(vaWithNoPVReferenceNorInlineVolumeSpec(va(true, fin, ann))),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaWithNoPVReferenceNorInlineVolumeSpec(va(true, fin, ann))),
						deleted(vaWithDetachError(vaWithNoPVReferenceNorInlineVolumeSpec(va(true, fin, ann)),
							"neither InlineCSIVolumeSource nor PersistentVolumeName specified in VA source"))),
					"status"),
			},
		},
		{
			name:           "detach VA with both PV and InlineCSIVolumeSource reference-> error",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        deleted(vaAddInlineSpec(va(true, fin, ann))),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(vaAddInlineSpec(va(true, fin, ann))),
						deleted(vaWithDetachError(vaAddInlineSpec(va(true, fin, ann)),
							"both InlineCSIVolumeSource and PersistentVolumeName specified in VA source"))),
					"status")},
		},
		{
			name:           "detach unknown node -> error",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        deleted(va(true, fin, nil)),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, nil)),
						deleted(vaWithDetachError(va(true, fin, nil),
							"csinode.storage.k8s.io \"node1\" not found"))), "status"),
			},
		},
		{
			name:           "detach unknown node -> use annotation",
			initialObjects: []runtime.Object{pvWithFinalizer()},
			addedVA:        deleted(va(true, fin, map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, fin, map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})),
						deleted(va(true /*attached*/, "",
							map[string]string{vaNodeIDAnnotation: "annotatedNodeID"}))), ""),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, "", map[string]string{vaNodeIDAnnotation: "annotatedNodeID"})),
						deleted(va(false /*attached*/, "",
							map[string]string{vaNodeIDAnnotation: "annotatedNodeID"}))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, "annotatedNodeID", noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "failed write with finalizer removal -> controller retries",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        deleted(va(true, fin, ann)),
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
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, fin, ann)),
						deleted(va(true, "", ann))), ""),
				// Saving error succeeds
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, fin, ann)),
						vaWithDetachError(deleted(va(true, fin, ann)),
							"could not mark as detached: volumeattachments.storage.k8s."+
								"io \"pv1-node1\" is forbidden: mock error")), "status"),
				// Second save of attached=false succeeds and the finalizer is subsequently deleted.
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, fin, ann)),
						deleted(va(true, "", ann))), ""),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						vaWithDetachError(deleted(va(true, "", ann)),
							"could not mark as detached: volumeattachments.storage.k8s."+
								"io \"pv1-node1\" is forbidden: mock error"),
						deleted(va(false, "", ann))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, detached, noMetadata, 0},
			},
		},
		{
			name:           "VolumeAttachment with GCEPersistentDiskVolumeSource marked for deletion -> successful detach",
			initialObjects: []runtime.Object{gcePDPVWithFinalizer(), csiNode()},
			addedVA:        deleted(va(true /*attached*/, fin /*finalizer*/, ann)),
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, fin, ann)),
						deleted(va(true /*attached*/, "", ann))), ""),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(
						deleted(va(true, "", ann)),
						deleted(va(false /*attached*/, "", ann))), "status"),
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
			name:           "VA deleted -> PV finalizer removed (GCE PD PV)",
			initialObjects: []runtime.Object{pvDeleted(gcePDPVWithFinalizer())},
			deletedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.MergePatchType, patch(pvDeleted(gcePDPVWithFinalizer()),
						pvDeleted(gcePDPV()))),
			},
		},
		{
			name:           "VA deleted -> PV finalizer not removed",
			initialObjects: []runtime.Object{pvDeleted(pv())},
			deletedVA:      va(false, "", nil),
		},
		{
			name:           "VA deleted -> PV finalizer not removed (GCE PD PV)",
			initialObjects: []runtime.Object{pvDeleted(gcePDPV())},
			deletedVA:      va(false, "", nil),
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
		{
			name:            "PV created by other CSI drivers or in-tree provisioners -> ignored",
			initialObjects:  []runtime.Object{},
			updatedPV:       pvWithDriverName("dummy"),
			expectedActions: []core.Action{},
		},
	}

	runTests(t, csiHandlerFactory, tests)
}

func TestVolumeAttachmentWithErrorCode(t *testing.T) {
	featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.MutableCSINodeAllocatableCount, true)

	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1",
		Resource: "volumeattachments",
	}

	var noMetadata map[string]string
	var noAttrs map[string]string
	var noSecrets map[string]string
	var notDetached = false
	var success error
	var readWrite = false

	test := testCase{
		name:           "CSI attach fails with gRPC error -> controller saves ErrorCode and retries",
		initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
		addedVA:        va(false, "", nil),
		expectedActions: []core.Action{
			core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
				types.MergePatchType, patch(va(false, "", nil), va(false, fin, ann))),

			// The CSI call fails, so the controller saves the error status.
			core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
				testPVName+"-"+testNodeName,
				types.MergePatchType, patch(va(false, fin, ann),
					vaWithAttachErrorAndCode(va(false, fin, ann), "rpc error: code = ResourceExhausted desc = mock rpc error", codes.ResourceExhausted)), "status"),

			// On retry, the controller reads the original VA again and tries to re-apply the finalizer/annotation.
			core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
				types.MergePatchType, patch(
					vaWithAttachErrorAndCode(va(false, "", nil), "rpc error: code = ResourceExhausted desc = mock rpc error", codes.ResourceExhausted),
					vaWithAttachErrorAndCode(va(false, fin, ann), "rpc error: code = ResourceExhausted desc = mock rpc error", codes.ResourceExhausted),
				)),

			// The CSI call succeeds now, and the controller clears the error and marks the VA as attached.
			core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
				testPVName+"-"+testNodeName,
				types.MergePatchType, patch(
					vaWithAttachErrorAndCode(va(false, fin, ann), "rpc error: code = ResourceExhausted desc = mock rpc error", codes.ResourceExhausted),
					va(true /*attached*/, fin, ann),
				),
				"status"),
		},
		expectedCSICalls: []csiCall{
			{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, status.Error(codes.ResourceExhausted, "mock rpc error"), notDetached, noMetadata, 0},
			{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
		},
	}

	runTests(t, csiHandlerFactory, []testCase{test})
}

func TestCSIHandlerReconcileVA(t *testing.T) {
	nID := map[string]string{
		vaNodeIDAnnotation: testNodeID,
	}
	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1",
		Resource: "volumeattachments",
	}
	tests := []testCase{
		// TODO: Add a test with volume type that supports migration
		// (Ref: https://github.com/kubernetes-csi/external-attacher/issues/247)
		{
			name: "va attached actual state not attached",
			initialObjects: []runtime.Object{
				va(true /*attached*/, fin /* Finalizer*/, nID /*annotations*/),
				pvWithFinalizer(),
				csiNode(),
			},
			listerResponse: map[string][]string{
				// Intentionally empty
			},
			expectedActions: []core.Action{
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(true /*attached*/, "", nil),
						va(true, "", nil)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, nil, nil, false, nil, false, nil, 0},
			},
		},
		{
			name: "va attached actual state attached",
			initialObjects: []runtime.Object{
				va(true /*attached*/, "" /*finalizer*/, nID /*annotations*/),
				pvWithFinalizer(),
			},
			listerResponse: map[string][]string{
				testVolumeHandle: {testNodeID},
			},
			expectedActions: []core.Action{
				// Intentionally empty
			},
		},
		{
			name: "va not attached actual state attached",
			initialObjects: []runtime.Object{
				deleted(va(false /*attached*/, "" /*finalizer*/, nID /*annotations*/)),
				pvWithFinalizer(),
			},
			listerResponse: map[string][]string{
				testVolumeHandle: {testNodeID},
			},
			expectedActions: []core.Action{},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, nil, nil, false, nil, true, nil, 0},
			},
		},
		{
			name:           "no volume attachments but existing lister response results in no action",
			initialObjects: []runtime.Object{},
			listerResponse: map[string][]string{
				testVolumeHandle: {testNodeID},
			},
			expectedActions: []core.Action{},
		},
	}
	runTests(t, csiHandlerFactory, tests)
}

func TestCSIHandlerReadOnly(t *testing.T) {
	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1",
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
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone,
					testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
		{
			name:           "read-only PV -> attached as read-write",
			initialObjects: []runtime.Object{pvReadOnly(pvWithFinalizer()), csiNode()},
			addedVA:        va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
			expectedActions: []core.Action{
				// Finalizer is saved first
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						va(false /*attached*/, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, fin, ann),
						va(true /*attached*/, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
		},
	}
	runTests(t, csiHandlerFactoryNoReadOnly, tests)
}

func TestMarkAsMigrated(t *testing.T) {
	t.Run("context has the migrated label for the migratable plugins", func(t *testing.T) {
		ctx := context.Background()
		migratedCtx := markAsMigrated(ctx, true)
		additionalInfo := migratedCtx.Value(connection.AdditionalInfoKey)
		if additionalInfo == nil {
			t.Errorf("test: %s, no migrated label found in the context", t.Name())
		}
		additionalInfoVal := additionalInfo.(connection.AdditionalInfo)
		migrated := additionalInfoVal.Migrated

		if migrated != "true" {
			t.Errorf("test: %s, expected: %v, got: %v", t.Name(), "true", migrated)
		}
	})
}
