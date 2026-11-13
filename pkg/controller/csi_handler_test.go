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
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/external-attacher/v4/pkg/attacher"
	"github.com/kubernetes-csi/external-attacher/v4/pkg/features"
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
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
	csitranslator "k8s.io/csi-translation-lib"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/ktesting"
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
		nil, /* no ControllerGetNodeInfo integration */
		nil,
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
		nil, /* no ControllerGetNodeInfo integration */
		nil,
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

func csiNodeWithNodeID(nodeID string) *storage.CSINode {
	node := csiNode()
	node.Spec.Drivers[0].NodeID = nodeID
	return node
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

func pvAddFinalizerJSON(pv *v1.PersistentVolume) []byte {
	p, err := addFinalizerPatch(pv.Finalizers, fin)
	if err != nil {
		klog.Background().Error(err, "Failed to create add-finalizer patch")
		return nil
	}
	return p
}

func pvRemoveFinalizerJSON(pv *v1.PersistentVolume) []byte {
	p, err := removeFinalizerPatch(pv.Finalizers, fin)
	if err != nil {
		klog.Background().Error(err, "Failed to create remove-finalizer patch")
		return nil
	}
	return p
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
					types.JSONPatchType, pvAddFinalizerJSON(pv())),
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
					types.JSONPatchType, pvAddFinalizerJSON(pv())),
				// Error is saved
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false /*attached*/, "" /*finalizer*/, nil /* annotations */),
						vaWithAttachError(va(false, "", nil),
							"could not add PersistentVolume finalizer: persistentvolume \"pv1\" is forbidden: Mock"+
								" error")), "status"),
				// Second PV Finalizer - succeeds
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.JSONPatchType, pvAddFinalizerJSON(pv())),
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
					types.JSONPatchType, pvRemoveFinalizerJSON(pvWithFinalizer())),
			},
		},
		{
			name:           "VA deleted -> PV finalizer removed (GCE PD PV)",
			initialObjects: []runtime.Object{pvDeleted(gcePDPVWithFinalizer())},
			deletedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.JSONPatchType, pvRemoveFinalizerJSON(gcePDPVWithFinalizer())),
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
					types.JSONPatchType, pvRemoveFinalizerJSON(pvWithFinalizer())),
			},
		},
		{
			name:           "PV finalizer removed -> other finalizers preserved",
			initialObjects: []runtime.Object{pvDeleted(pvWithFinalizers(pvWithFinalizer(), "foo/bar", "bar/baz"))},
			deletedVA:      va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.JSONPatchType, pvRemoveFinalizerJSON(pvWithFinalizers(pvWithFinalizer(), "foo/bar", "bar/baz"))),
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
					types.JSONPatchType, pvRemoveFinalizerJSON(pvWithFinalizer())),
				// This one fails too
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.JSONPatchType, pvRemoveFinalizerJSON(pvWithFinalizer())),
				// This one succeeds
				core.NewPatchAction(pvGroupResourceVersion, metav1.NamespaceNone, testPVName,
					types.JSONPatchType, pvRemoveFinalizerJSON(pvWithFinalizer())),
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

func TestCSIHandlerRecordsProcessing(t *testing.T) {
	nodeInfo := &fakeNodeInfo{}
	// The handler factory resets nodeInfo, so each case starts from an empty record.
	expectProcessing := func(want ...string) func(*testing.T, testCase) {
		return func(t *testing.T, _ testCase) {
			if !slices.Equal(nodeInfo.recorded, want) {
				t.Errorf("recorded volume operations: want %v, got %v", want, nodeInfo.recorded)
			}
		}
	}

	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1",
		Resource: "volumeattachments",
	}

	var noMetadata map[string]string
	var noAttrs map[string]string
	var noSecrets map[string]string
	var success error
	var notDetached = false
	var readWrite = false
	const gceNodeID = "projects/test-project/zones/testZone/instances/test-node"
	gceAnn := map[string]string{vaNodeIDAnnotation: gceNodeID}

	tests := []testCase{
		{
			name:           "attach of a CSI volume",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, "", nil), va(false, fin, ann))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, ann), va(true, fin, ann)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
			additionalCheck: expectProcessing(testNodeID + "/" + testVolumeHandle),
		},
		{
			name:           "detach of a CSI volume",
			initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
			addedVA:        deleted(va(true, fin, ann)),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, fin, ann)), deleted(va(true, "", ann)))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(deleted(va(true, "", ann)), deleted(va(false, "", ann))), "status"),
			},
			expectedCSICalls: []csiCall{
				{"detach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, success, false, noMetadata, 0},
			},
			additionalCheck: expectProcessing(testNodeID + "/" + testVolumeHandle),
		},
		{
			// The driver knows the volume by the repaired handle, so that is what it
			// reports in ControllerGetNodeInfo.
			name:           "attach of a migrated in-tree volume",
			initialObjects: []runtime.Object{pvWithFinalizers(gcePDPV(), fin), csiNodeWithNodeID(gceNodeID)},
			addedVA:        va(false, "", nil),
			expectedActions: []core.Action{
				core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, "", nil), va(false, fin, gceAnn))),
				core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
					types.MergePatchType, patch(va(false, fin, gceAnn), va(true, fin, gceAnn)), "status"),
			},
			expectedCSICalls: []csiCall{
				{"attach", "projects/UNSPECIFIED/zones/testZone/disks/testpd", gceNodeID, map[string]string{"partition": ""}, noSecrets, readWrite, success, notDetached, noMetadata, 0},
			},
			additionalCheck: expectProcessing(gceNodeID + "/projects/test-project/zones/testZone/disks/testpd"),
		},
	}

	runTests(t, csiHandlerFactoryWithNodeInfo(nodeInfo), tests)
}

func TestCSIHandlerNodeInfoRefresh(t *testing.T) {
	vaGroupResourceVersion := schema.GroupVersionResource{
		Group:    storage.GroupName,
		Version:  "v1",
		Resource: "volumeattachments",
	}

	var noMetadata map[string]string
	var noAttrs map[string]string
	var noSecrets map[string]string
	var notDetached = false
	var readWrite = false
	var noError error
	exhausted := status.Error(codes.ResourceExhausted, "mock rpc error")
	internal := status.Error(codes.Internal, "mock rpc error")

	tests := []struct {
		name        string
		err         error
		refreshErr  error
		wantRefresh bool
	}{
		{name: "attach success", err: noError},
		{name: "other attach error", err: internal},
		{name: "RESOURCE_EXHAUSTED", err: exhausted, wantRefresh: true},
		{name: "RESOURCE_EXHAUSTED with failing refresh", err: exhausted, refreshErr: errors.New("mock refresh error"), wantRefresh: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodeInfo := &fakeNodeInfo{refreshErr: test.refreshErr}
			tc := testCase{
				name:           test.name,
				initialObjects: []runtime.Object{pvWithFinalizer(), csiNode()},
				addedVA:        va(false, "", nil),
				expectedActions: []core.Action{
					core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
						types.MergePatchType, patch(va(false, "", nil), va(false, fin, ann))),
					core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
						types.MergePatchType, patch(va(false, fin, ann), va(true, fin, ann)), "status"),
				},
				expectedCSICalls: []csiCall{
					{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, noError, notDetached, noMetadata, 0},
				},
				additionalCheck: func(t *testing.T, _ testCase) {
					if got := len(nodeInfo.refreshed); (got > 0) != test.wantRefresh {
						t.Fatalf("node info refreshes: want %t, got %v", test.wantRefresh, nodeInfo.refreshed)
					}
					if !test.wantRefresh {
						return
					}
					if !slices.Equal(nodeInfo.refreshed, []string{testNodeName}) {
						t.Errorf("refreshed nodes: want %v, got %v", []string{testNodeName}, nodeInfo.refreshed)
					}
					// The corrected count must be published before the attach error is reported.
					statusPatch := slices.IndexFunc(nodeInfo.client.Actions(), func(a core.Action) bool {
						return a.GetVerb() == "patch" && a.GetSubresource() == "status"
					})
					if statusPatch < 0 || statusPatch < nodeInfo.actionsBeforeRefresh {
						t.Errorf("attach error reported at action %d, not after the node info refresh (%d actions)", statusPatch, nodeInfo.actionsBeforeRefresh)
					}
				},
			}

			if test.err != noError {
				withAttachError := func(va *storage.VolumeAttachment) *storage.VolumeAttachment {
					return vaWithAttachErrorAndCode(va, test.err.Error(), status.Code(test.err))
				}
				// The failed attach is retried and succeeds, so the test terminates.
				// Handler refreshes only the first attempt.
				tc.expectedActions = []core.Action{
					core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
						types.MergePatchType, patch(va(false, "", nil), va(false, fin, ann))),
					core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
						types.MergePatchType, patch(va(false, fin, ann), withAttachError(va(false, fin, ann))), "status"),
					core.NewPatchAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
						types.MergePatchType, patch(withAttachError(va(false, "", nil)), withAttachError(va(false, fin, ann)))),
					core.NewPatchSubresourceAction(vaGroupResourceVersion, metav1.NamespaceNone, testPVName+"-"+testNodeName,
						types.MergePatchType, patch(withAttachError(va(false, fin, ann)), va(true, fin, ann)), "status"),
				}
				tc.expectedCSICalls = []csiCall{
					{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, test.err, notDetached, noMetadata, 0},
					{"attach", testVolumeHandle, testNodeID, noAttrs, noSecrets, readWrite, noError, notDetached, noMetadata, 0},
				}
			}

			runTests(t, csiHandlerFactoryWithNodeInfo(nodeInfo), []testCase{tc})
		})
	}
}

// fakeNodeInfo records how the VolumeAttachment worker uses the ControllerGetNodeInfo flow.
type fakeNodeInfo struct {
	mu sync.Mutex

	client     *fake.Clientset
	recorded   []string
	refreshed  []string
	refreshErr error

	// Number of client actions issued when the last refresh was requested.
	actionsBeforeRefresh int
}

func (f *fakeNodeInfo) RecordProcessing(nodeID, volumeID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recorded = append(f.recorded, nodeID+"/"+volumeID)
}

func (f *fakeNodeInfo) RequestUpdate(_ context.Context, nodeName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshed = append(f.refreshed, nodeName)
	if f.client != nil {
		f.actionsBeforeRefresh = len(f.client.Actions())
	}
	return f.refreshErr
}

// reset prepares the fake for a new test case, which runTests starts by
// constructing a new handler.
func (f *fakeNodeInfo) reset(client *fake.Clientset) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.client = client
	f.recorded = nil
	f.refreshed = nil
	f.actionsBeforeRefresh = 0
}

func csiHandlerFactoryWithNodeInfo(nodeInfo *fakeNodeInfo) handlerFactory {
	return func(client kubernetes.Interface, informerFactory informers.SharedInformerFactory, csi attacher.Attacher, lister VolumeLister) Handler {
		fakeClient, _ := client.(*fake.Clientset)
		nodeInfo.reset(fakeClient)
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
			nodeInfo,
			nodeInfo,
		)
	}
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

func TestVolumeSourceResolver(t *testing.T) {
	pvLister := func(pvs ...*v1.PersistentVolume) corelisters.PersistentVolumeLister {
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
		for _, pv := range pvs {
			if err := indexer.Add(pv); err != nil {
				t.Fatalf("add PV to indexer: %v", err)
			}
		}
		return corelisters.NewPersistentVolumeLister(indexer)
	}
	resolver := func(pvs ...*v1.PersistentVolume) volumeSourceResolver {
		return volumeSourceResolver{pvLister: pvLister(pvs...), translator: csitranslator.New()}
	}
	inlineSpecWithoutCSI := func() *storage.VolumeAttachment {
		va := va(false /*attached*/, "" /*finalizer*/, nil /*annotations*/)
		va.Spec.Source.PersistentVolumeName = nil
		va.Spec.Source.InlineVolumeSpec = &v1.PersistentVolumeSpec{}
		return va
	}
	nonCSIPV := pv()
	nonCSIPV.Spec.PersistentVolumeSource = v1.PersistentVolumeSource{
		HostPath: &v1.HostPathVolumeSource{Path: "/tmp"},
	}

	// Handle generated by TranslateInTreePVToCSI for gcePDPV()
	const gceHandle = "projects/UNSPECIFIED/zones/testZone/disks/testpd"

	tests := []struct {
		name         string
		resolver     volumeSourceResolver
		va           *storage.VolumeAttachment
		wantPV       bool
		wantHandle   string
		wantMigrated bool
		wantErr      string
	}{
		{
			name:       "CSI PV",
			resolver:   resolver(pv()),
			va:         va(false, "", nil),
			wantPV:     true,
			wantHandle: testVolumeHandle,
		},
		{
			name:         "migrated in-tree PV",
			resolver:     resolver(gcePDPV()),
			va:           va(false, "", nil),
			wantPV:       true,
			wantHandle:   gceHandle,
			wantMigrated: true,
		},
		{
			name:         "inline volume spec",
			resolver:     resolver(),
			va:           vaWithInlineSpec(va(false, "", nil)),
			wantHandle:   testVolumeHandle,
			wantMigrated: true,
		},
		{
			name:     "inline volume spec without CSI source",
			resolver: resolver(),
			va:       inlineSpecWithoutCSI(),
			wantErr:  "inline volume spec contains nil CSI source",
		},
		{
			name:     "both PV and inline volume spec",
			resolver: resolver(pv()),
			va:       vaAddInlineSpec(va(false, "", nil)),
			wantErr:  "both InlineCSIVolumeSource and PersistentVolumeName",
		},
		{
			name:     "neither PV nor inline volume spec",
			resolver: resolver(),
			va:       vaWithNoPVReferenceNorInlineVolumeSpec(va(false, "", nil)),
			wantErr:  "neither InlineCSIVolumeSource nor PersistentVolumeName",
		},
		{
			name:     "PV not found",
			resolver: resolver(),
			va:       va(false, "", nil),
			wantErr:  "not found",
		},
		{
			name:     "PV without CSI source and not migratable",
			resolver: resolver(nonCSIPV),
			va:       va(false, "", nil),
			wantErr:  "pv spec contained non-csi source that was not migrated",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := ktesting.NewTestContext(t)
			src, err := test.resolver.resolve(logger, test.va)

			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve failed: %v", err)
			}
			if got := src.spec.CSI.VolumeHandle; got != test.wantHandle {
				t.Errorf("volumeHandle: want %q, got %q", test.wantHandle, got)
			}
			if src.migrated != test.wantMigrated {
				t.Errorf("migrated: want %t, got %t", test.wantMigrated, src.migrated)
			}
			if gotPV := src.pv != nil; gotPV != test.wantPV {
				t.Errorf("pv: want %t, got %t", test.wantPV, gotPV)
			}
			if src.migrated && src.pv != nil && src.pv.Spec.GCEPersistentDisk == nil {
				t.Errorf("expected the original in-tree PV, got a translated one")
			}
		})
	}
}

type recordingTranslator struct {
	AttacherCSITranslator
	repaired []string
}

func (r *recordingTranslator) RepairVolumeHandle(pluginName, volumeHandle, nodeID string) (string, error) {
	r.repaired = append(r.repaired, volumeHandle)
	return r.AttacherCSITranslator.RepairVolumeHandle(pluginName, volumeHandle, nodeID)
}

// newReconcileTestHandler returns a csiHandler that is ready for ReconcileVA,
// with the given objects preloaded into the informer stores.
func newReconcileTestHandler(t *testing.T, translator AttacherCSITranslator, lister VolumeLister, objects ...runtime.Object) *csiHandler {
	t.Helper()
	client := fake.NewSimpleClientset()
	factory := informers.NewSharedInformerFactory(client, time.Hour)
	pvInformer := factory.Core().V1().PersistentVolumes()
	vaInformer := factory.Storage().V1().VolumeAttachments()
	csiNodeInformer := factory.Storage().V1().CSINodes()
	for _, obj := range objects {
		var err error
		switch obj := obj.(type) {
		case *v1.PersistentVolume:
			err = pvInformer.Informer().GetStore().Add(obj)
		case *storage.VolumeAttachment:
			err = vaInformer.Informer().GetStore().Add(obj)
		case *storage.CSINode:
			err = csiNodeInformer.Informer().GetStore().Add(obj)
		default:
			t.Fatalf("unsupported initial object type: %T", obj)
		}
		if err != nil {
			t.Fatalf("add initial object: %v", err)
		}
	}

	h := NewCSIHandler(
		client,
		testAttacherName,
		nil, /* attacher is not used by ReconcileVA */
		lister,
		pvInformer.Lister(),
		csiNodeInformer.Lister(),
		vaInformer.Lister(),
		&timeout,
		true,  /* supports PUBLISH_READONLY */
		false, /* does not support SINGLE_NODE_MULTI_WRITER access mode */
		translator,
		defaultFSType,
		nil, /* no ControllerGetNodeInfo integration */
		nil,
	).(*csiHandler)
	h.Init(workqueue.NewTypedRateLimitingQueue[string](workqueue.DefaultTypedControllerRateLimiter[string]()), nil)
	return h
}

func TestReconcileVAMigratedVolumeHandle(t *testing.T) {
	const (
		gceDriver         = "pd.csi.storage.gke.io"
		gceRawHandle      = "projects/UNSPECIFIED/zones/testZone/disks/testpd"
		gceRepairedHandle = "projects/test-project/zones/testZone/disks/testpd"
		gceNodeID         = "projects/test-project/zones/testZone/instances/test-node"
	)
	inlineVA := func() *storage.VolumeAttachment {
		va := va(true /*attached*/, "" /*finalizer*/, nil /*annotations*/)
		va.Spec.Source.PersistentVolumeName = nil
		va.Spec.Source.InlineVolumeSpec = &v1.PersistentVolumeSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       gceDriver,
					VolumeHandle: gceRawHandle,
				},
			},
		}
		return va
	}

	tests := []struct {
		name string
		// Initial objects, preloaded into the informer stores
		objects []runtime.Object
		// What ListVolumes reports
		published map[string][]string
		// Handles passed to RepairVolumeHandle
		wantRepaired []string
		// Whether the VA is enqueued for a forced attach/detach
		wantForceSynced bool
	}{
		{
			name:         "migrated PV is matched by the repaired volume handle",
			objects:      []runtime.Object{va(true, "", nil), gcePDPV(), csiNodeWithNodeID(gceNodeID)},
			published:    map[string][]string{gceRepairedHandle: {gceNodeID}},
			wantRepaired: []string{gceRawHandle},
		},
		{
			name:            "migrated PV is not matched by the unrepaired volume handle",
			objects:         []runtime.Object{va(true, "", nil), gcePDPV(), csiNodeWithNodeID(gceNodeID)},
			published:       map[string][]string{gceRawHandle: {gceNodeID}},
			wantRepaired:    []string{gceRawHandle},
			wantForceSynced: true,
		},
		{
			name:      "CSI PV is never repaired",
			objects:   []runtime.Object{va(true, "", nil), pv(), csiNode()},
			published: map[string][]string{testVolumeHandle: {testNodeID}},
		},
		{
			name:         "migrated inline volume is matched by the repaired volume handle",
			objects:      []runtime.Object{inlineVA(), csiNodeWithNodeID(gceNodeID)},
			published:    map[string][]string{gceRepairedHandle: {gceNodeID}},
			wantRepaired: []string{gceRawHandle},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, ctx := ktesting.NewTestContext(t)
			translator := &recordingTranslator{AttacherCSITranslator: csitranslator.New()}
			lister := &fakeLister{t: t, publishedNodes: test.published}
			h := newReconcileTestHandler(t, translator, lister, test.objects...)

			if err := h.ReconcileVA(klog.NewContext(ctx, logger)); err != nil {
				t.Fatalf("ReconcileVA failed: %v", err)
			}
			if !slices.Equal(translator.repaired, test.wantRepaired) {
				t.Errorf("RepairVolumeHandle calls: want %v, got %v", test.wantRepaired, translator.repaired)
			}
			forceSynced := h.consumeForceSync(testPVName + "-" + testNodeName)
			if forceSynced != test.wantForceSynced {
				t.Errorf("forceSync: want %t, got %t", test.wantForceSynced, forceSynced)
			}
			if got, want := h.vaQueue.Len(), boolToInt(test.wantForceSynced); got != want {
				t.Errorf("VA queue length: want %d, got %d", want, got)
			}
		})
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
