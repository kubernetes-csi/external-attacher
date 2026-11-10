/*
Copyright 2026 The Kubernetes Authors.

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
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/external-attacher/v4/pkg/attacher"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
)

type fakeNodeInfoProvider struct {
	resp *csi.ControllerGetNodeInfoResponse
	err  error
}

func (p *fakeNodeInfoProvider) GetNodeInfo(ctx context.Context, nodeID string) (*csi.ControllerGetNodeInfoResponse, error) {
	return p.resp, p.err
}

func newNodeTestHandler(t *testing.T, client *fake.Clientset, provider attacher.NodeInfoProvider) (*nodeCSIHandler, *record.FakeRecorder) {
	t.Helper()
	factory := informers.NewSharedInformerFactory(client, 0)
	recorder := record.NewFakeRecorder(10)
	h := NewNodeCSIHandler(
		testAttacherName,
		client,
		provider,
		recorder,
		factory.Storage().V1().VolumeAttachments().TypedInformer(),
		factory.Core().V1().PersistentVolumes(),
		nil,
	)
	return h.(*nodeCSIHandler), recorder
}

func testNode(uid string) *corev1.Node {
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: testNodeName, UID: types.UID(uid)}}
}

func testCSINode(nodeUID string, drivers ...storagev1.CSINodeDriver) *storagev1.CSINode {
	return &storagev1.CSINode{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testNodeName,
			UID:             "csinode-uid",
			ResourceVersion: "1",
			OwnerReferences: []metav1.OwnerReference{{Kind: "Node", UID: types.UID(nodeUID)}},
		},
		Spec: storagev1.CSINodeSpec{
			Drivers: drivers,
			DriverRegistrations: []storagev1.CSINodeDriverRegistration{
				{Name: testAttacherName, NodeID: testNodeID},
			},
		},
	}
}

func csinodeWrites(actions []k8stesting.Action) int {
	writes := 0
	for _, a := range actions {
		if a.GetVerb() != "patch" && a.GetVerb() != "update" {
			continue
		}
		if _, ok := a.(k8stesting.PatchAction); ok && a.GetResource().Resource == "csinodes" {
			writes++
		}
	}
	return writes
}

func drainEvents(recorder *record.FakeRecorder) []string {
	var events []string
	for {
		select {
		case e := <-recorder.Events:
			events = append(events, e)
		default:
			return events
		}
	}
}

func TestSyncNodeInitialRegistration(t *testing.T) {
	node := testNode("node-uid")
	csiNode := testCSINode("node-uid")
	client := fake.NewClientset(node, csiNode)
	provider := &fakeNodeInfoProvider{resp: &csi.ControllerGetNodeInfoResponse{
		MaxVolumesPerNode: 10,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{"topology.kubernetes.io/zone": "zone-a"},
		},
	}}
	h, recorder := newNodeTestHandler(t, client, provider)

	if err := h.SyncNode(t.Context(), csiNode, testNodeID, false); err != nil {
		t.Fatalf("SyncNode: %v", err)
	}

	updated, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSINode: %v", err)
	}
	if len(updated.Spec.Drivers) != 1 {
		t.Fatalf("expected 1 driver entry, got %v", updated.Spec.Drivers)
	}
	driver := updated.Spec.Drivers[0]
	if driver.Name != testAttacherName || driver.NodeID != testNodeID {
		t.Errorf("unexpected driver entry: %+v", driver)
	}
	if want := []string{"topology.kubernetes.io/zone"}; !slices.Equal(driver.TopologyKeys, want) {
		t.Errorf("topology keys: got %v, want %v", driver.TopologyKeys, want)
	}
	if driver.Allocatable == nil || ptr.Deref(driver.Allocatable.Count, 0) != 10 {
		t.Errorf("allocatable: got %+v, want count 10", driver.Allocatable)
	}

	updatedNode, err := client.CoreV1().Nodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get Node: %v", err)
	}
	if got := updatedNode.Labels["topology.kubernetes.io/zone"]; got != "zone-a" {
		t.Errorf("node label: got %q, want %q", got, "zone-a")
	}

	events := drainEvents(recorder)
	if len(events) != 1 || !strings.Contains(events[0], "CSINodeRegistered") {
		t.Errorf("unexpected events: %v", events)
	}
	if !strings.Contains(events[0], fmt.Sprintf("%q", testNodeID)) || !strings.Contains(events[0], "allocatableCount=10") {
		t.Errorf("unexpected event message: %v", events)
	}
}

func TestSyncNodeRefresh(t *testing.T) {
	testcases := []struct {
		name      string
		newCount  int64
		wantWrite bool
		wantCount *int32 // nil means no allocatable
		wantEvent string // substring of the expected event message
	}{
		{name: "unchanged", newCount: 10, wantWrite: false, wantCount: ptr.To[int32](10)},
		{name: "changed", newCount: 8, wantWrite: true, wantCount: ptr.To[int32](8), wantEvent: "allocatableCount=8"},
		{name: "limit removed", newCount: 0, wantWrite: true, wantCount: nil, wantEvent: "allocatableCount=unlimited"},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			node := testNode("node-uid")
			client := fake.NewClientset(node, testCSINode("node-uid"))
			provider := &fakeNodeInfoProvider{resp: &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10}}
			h, recorder := newNodeTestHandler(t, client, provider)
			// Register first, external-attacher can only remove fields it owns.
			if err := h.SyncNode(t.Context(), testCSINode("node-uid"), testNodeID, false); err != nil {
				t.Fatalf("register: %v", err)
			}
			csiNode, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("get CSINode: %v", err)
			}
			provider.resp = &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: tc.newCount}
			client.ClearActions()
			drainEvents(recorder)

			if err := h.SyncNode(t.Context(), csiNode, testNodeID, true); err != nil {
				t.Fatalf("refresh: %v", err)
			}

			if writes := csinodeWrites(client.Actions()); (writes > 0) != tc.wantWrite {
				t.Errorf("CSINode writes: got %d, want write %v", writes, tc.wantWrite)
			}
			events := drainEvents(recorder)
			if tc.wantWrite != (len(events) > 0) {
				t.Errorf("events: got %v, want event %v", events, tc.wantWrite)
			}
			for _, e := range events {
				if !strings.Contains(e, "CSINodeInfoUpdated") {
					t.Errorf("unexpected reason in event: %v", e)
				}
			}
			if tc.wantEvent != "" && (len(events) == 0 || !strings.Contains(events[0], tc.wantEvent)) {
				t.Errorf("events: got %v, want one containing %q", events, tc.wantEvent)
			}

			updated, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("get CSINode: %v", err)
			}
			if len(updated.Spec.Drivers) != 1 {
				t.Fatalf("expected 1 driver entry, got %v", updated.Spec.Drivers)
			}
			allocatable := updated.Spec.Drivers[0].Allocatable
			if tc.wantCount == nil && allocatable != nil {
				t.Errorf("allocatable: got %+v, want unset", allocatable)
			}
			if tc.wantCount != nil && (allocatable == nil || !ptr.Equal(allocatable.Count, tc.wantCount)) {
				t.Errorf("allocatable: got %+v, want count %d", allocatable, *tc.wantCount)
			}
		})
	}
}

func TestSyncNodeRefreshClaimsOwnership(t *testing.T) {
	// Entry created by kubelet: the desired state already matches, but external-attacher
	// owns none of it, so the first refresh must apply to claim the fields.
	csiNode := testCSINode("node-uid", storagev1.CSINodeDriver{
		Name:         testAttacherName,
		NodeID:       testNodeID,
		TopologyKeys: []string{"topology.kubernetes.io/zone"},
		Allocatable:  &storagev1.VolumeNodeResources{Count: ptr.To[int32](10)},
	})
	client := fake.NewClientset(testNode("node-uid"))
	if _, err := client.StorageV1().CSINodes().Create(t.Context(), csiNode, metav1.CreateOptions{FieldManager: "kubelet"}); err != nil {
		t.Fatalf("create CSINode: %v", err)
	}
	provider := &fakeNodeInfoProvider{resp: &csi.ControllerGetNodeInfoResponse{
		MaxVolumesPerNode: 10,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{"topology.kubernetes.io/zone": "zone-a"},
		},
	}}
	h, recorder := newNodeTestHandler(t, client, provider)

	snapshot := func() *storagev1.CSINode {
		t.Helper()
		snapshot, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get CSINode: %v", err)
		}
		return snapshot
	}

	client.ClearActions()
	if err := h.SyncNode(t.Context(), snapshot(), testNodeID, true); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if writes := csinodeWrites(client.Actions()); writes != 1 {
		t.Errorf("CSINode writes: got %d, want 1", writes)
	}
	if events := drainEvents(recorder); len(events) != 1 {
		t.Errorf("events: got %v, want 1", events)
	}

	// We own the fields now, an identical refresh must be a no-op.
	client.ClearActions()
	if err := h.SyncNode(t.Context(), snapshot(), testNodeID, true); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if writes := csinodeWrites(client.Actions()); writes != 0 {
		t.Errorf("CSINode writes: got %d, want 0", writes)
	}
	if events := drainEvents(recorder); len(events) != 0 {
		t.Errorf("unexpected events: %v", events)
	}
}

func TestSyncNodeRefreshKeepsTopologyKeys(t *testing.T) {
	node := testNode("node-uid")
	client := fake.NewClientset(node, testCSINode("node-uid"))
	provider := &fakeNodeInfoProvider{resp: &csi.ControllerGetNodeInfoResponse{
		MaxVolumesPerNode: 10,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{"topology.kubernetes.io/zone": "zone-a"},
		},
	}}
	h, _ := newNodeTestHandler(t, client, provider)
	if err := h.SyncNode(t.Context(), testCSINode("node-uid"), testNodeID, false); err != nil {
		t.Fatalf("register: %v", err)
	}
	csiNode, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSINode: %v", err)
	}

	// Refresh with a new count and a different topology. The count must be updated,
	// the topology keys must be kept.
	provider.resp = &csi.ControllerGetNodeInfoResponse{
		MaxVolumesPerNode: 8,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{"other": "value"},
		},
	}
	if err := h.SyncNode(t.Context(), csiNode, testNodeID, true); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	updated, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSINode: %v", err)
	}
	if want := []string{"topology.kubernetes.io/zone"}; !slices.Equal(updated.Spec.Drivers[0].TopologyKeys, want) {
		t.Errorf("topology keys: got %v, want %v", updated.Spec.Drivers[0].TopologyKeys, want)
	}
	if got := ptr.Deref(updated.Spec.Drivers[0].Allocatable.Count, 0); got != 8 {
		t.Errorf("allocatable count: got %d, want 8", got)
	}
}

func TestSyncNodeGetNodeInfoFailed(t *testing.T) {
	node := testNode("node-uid")
	csiNode := testCSINode("node-uid")
	client := fake.NewClientset(node, csiNode)
	provider := &fakeNodeInfoProvider{err: errors.New("boom")}
	h, recorder := newNodeTestHandler(t, client, provider)

	err := h.SyncNode(t.Context(), csiNode, testNodeID, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if writes := csinodeWrites(client.Actions()); writes != 0 {
		t.Errorf("unexpected CSINode writes: %d", writes)
	}
	// Failures are reported by NodeController, see TestProcessNodeEvents.
	if events := drainEvents(recorder); len(events) != 0 {
		t.Errorf("unexpected events: %v", events)
	}
}
