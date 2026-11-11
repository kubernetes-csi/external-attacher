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
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	csitranslator "k8s.io/csi-translation-lib"
	"k8s.io/utils/ptr"
)

const (
	nodeOwnerKind = "Node"
	zoneKey       = "topology.kubernetes.io/zone"
	regionKey     = "topology.kubernetes.io/region"
)

// testNodeInfoProvider is a ControllerGetNodeInfo implementation whose response
// the test controls, including from within the call itself.
type testNodeInfoProvider struct {
	mu      sync.Mutex
	resp    *csi.ControllerGetNodeInfoResponse
	err     error
	calls   []string
	onCall  func(nodeID string)
	blockOn context.Context
}

func (p *testNodeInfoProvider) GetNodeInfo(ctx context.Context, nodeID string) (*csi.ControllerGetNodeInfoResponse, error) {
	p.mu.Lock()
	p.calls = append(p.calls, nodeID)
	onCall := p.onCall
	blockOn := p.blockOn
	resp, err := p.resp, p.err
	p.mu.Unlock()

	if onCall != nil {
		onCall(nodeID)
	}
	if blockOn != nil {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return resp, err
}

func (p *testNodeInfoProvider) set(resp *csi.ControllerGetNodeInfoResponse, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resp, p.err = resp, err
}

func (p *testNodeInfoProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.calls)
}

func publishedVolumes(ids ...string) []*csi.ControllerGetNodeInfoResponse_PublishedVolume {
	var volumes []*csi.ControllerGetNodeInfoResponse_PublishedVolume
	for _, id := range ids {
		volumes = append(volumes, &csi.ControllerGetNodeInfoResponse_PublishedVolume{VolumeId: id})
	}
	return volumes
}

func topology(segments map[string]string) *csi.Topology {
	return &csi.Topology{Segments: segments}
}

func testNode(nodeUID string, labels map[string]string) *corev1.Node {
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{
		Name:   testNodeName,
		UID:    types.UID(nodeUID),
		Labels: labels,
	}}
}

// testCSINode returns a CSINode owned by the given Node, with the driver
// registered by kubelet and the given completed Spec.Drivers entries.
func testCSINode(nodeUID string, drivers ...storagev1.CSINodeDriver) *storagev1.CSINode {
	return &storagev1.CSINode{
		ObjectMeta: metav1.ObjectMeta{
			Name:            testNodeName,
			UID:             "csinode-uid",
			ResourceVersion: "1",
			OwnerReferences: []metav1.OwnerReference{{Kind: nodeOwnerKind, UID: types.UID(nodeUID)}},
		},
		Spec: storagev1.CSINodeSpec{
			Drivers: drivers,
			DriverRegistrations: []storagev1.CSINodeDriverRegistration{
				{Name: testAttacherName, NodeID: testNodeID},
			},
		},
	}
}

func allocatableEqual(a, b *storagev1.VolumeNodeResources) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	return a == nil || ptr.Equal(a.Count, b.Count)
}

func installedDriver(topoKeys []string, count *int32) storagev1.CSINodeDriver {
	driver := storagev1.CSINodeDriver{Name: testAttacherName, NodeID: testNodeID, TopologyKeys: topoKeys}
	if count != nil {
		driver.Allocatable = &storagev1.VolumeNodeResources{Count: count}
	}
	return driver
}

func testPV(name, handle string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{Driver: testAttacherName, VolumeHandle: handle},
			},
		},
	}
}

func inlineVA() *storagev1.VolumeAttachment {
	attachment := va(true, "", nil)
	attachment.Spec.Source.PersistentVolumeName = nil
	return vaAddInlineSpec(attachment)
}

func testVA(attacher, pvName, nodeName string) *storagev1.VolumeAttachment {
	return createVolumeAttachment(attacher, pvName, nodeName, true, "", nil)
}

func newNodeTestHandler(t *testing.T, translator AttacherCSITranslator, objs ...runtime.Object) (*nodeCSIHandler, *fake.Clientset, *record.FakeRecorder, *testNodeInfoProvider) {
	t.Helper()
	if translator == nil {
		translator = csitranslator.New()
	}
	client := fake.NewClientset(objs...)
	factory := informers.NewSharedInformerFactory(client, 0)
	recorder := record.NewFakeRecorder(100)
	provider := &testNodeInfoProvider{}
	h := NewNodeCSIHandler(
		testAttacherName,
		client,
		provider,
		time.Minute,
		recorder,
		factory.Storage().V1().VolumeAttachments().TypedInformer(),
		factory.Core().V1().PersistentVolumes(),
		translator,
	).(*nodeCSIHandler)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), h.HasSynced) {
		t.Fatal("cannot sync informers")
	}
	return h, client, recorder, provider
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

func writeCount(actions []k8stesting.Action, resource string) int {
	writes := 0
	for _, a := range actions {
		if _, ok := a.(k8stesting.PatchAction); ok && a.GetResource().Resource == resource {
			writes++
		}
	}
	return writes
}

type syncNodeTest struct {
	name string

	node    *corev1.Node // nil for a missing Node object
	csiNode *storagev1.CSINode
	objects []runtime.Object // PVs, VAs and anything else needed

	translator  AttacherCSITranslator
	resp        *csi.ControllerGetNodeInfoResponse
	providerErr error
	block       bool // block until the context expires
	refresh     bool
	timeout     time.Duration
	reactor     k8stesting.ReactionFunc

	wantErr         string
	wantDriver      *storagev1.CSINodeDriver // nil means no driver entry expected
	wantOtherDriver bool                     // an entry of another driver must be preserved
	wantNodeLabels  map[string]string
	wantNodeWrites  int
	wantCalls       int
	wantEvent       string // substring of the single expected event, empty means none
}

func (tc *syncNodeTest) allObjects() []runtime.Object {
	objs := slices.Clone(tc.objects)
	if tc.node != nil {
		objs = append(objs, tc.node)
	}
	return append(objs, tc.csiNode)
}

func runSyncNodeTest(t *testing.T, tc syncNodeTest) {
	t.Helper()
	h, client, recorder, provider := newNodeTestHandler(t, tc.translator, tc.allObjects()...)
	provider.set(tc.resp, tc.providerErr)
	if tc.block {
		provider.blockOn = context.Background()
	}
	if tc.timeout > 0 {
		h.timeout = tc.timeout
	}
	if tc.reactor != nil {
		client.PrependReactor("*", "*", tc.reactor)
	}

	err := h.SyncNode(t.Context(), tc.csiNode, testNodeID, tc.refresh)

	if tc.wantErr == "" && err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.wantErr != "" {
		if err == nil {
			t.Fatalf("expected error containing %q, got success", tc.wantErr)
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("error: got %q, want it to contain %q", err, tc.wantErr)
		}
	}

	if tc.wantCalls >= 0 && provider.callCount() != tc.wantCalls {
		t.Errorf("ControllerGetNodeInfo calls: got %d, want %d", provider.callCount(), tc.wantCalls)
	}

	updated, getErr := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if getErr != nil {
		t.Fatalf("get CSINode: %v", getErr)
	}
	var driver *storagev1.CSINodeDriver
	for i := range updated.Spec.Drivers {
		if updated.Spec.Drivers[i].Name == testAttacherName {
			driver = &updated.Spec.Drivers[i]
		}
	}
	if tc.wantDriver == nil && driver != nil {
		t.Errorf("unexpected driver entry: %+v", *driver)
	}
	if tc.wantDriver != nil {
		if driver == nil {
			t.Fatalf("missing driver entry, spec.drivers: %+v", updated.Spec.Drivers)
		}
		if !slices.Equal(driver.TopologyKeys, tc.wantDriver.TopologyKeys) {
			t.Errorf("topology keys: got %v, want %v", driver.TopologyKeys, tc.wantDriver.TopologyKeys)
		}
		if !allocatableEqual(driver.Allocatable, tc.wantDriver.Allocatable) {
			t.Errorf("allocatable: got %+v, want %+v", driver.Allocatable, tc.wantDriver.Allocatable)
		}
		if driver.NodeID != tc.wantDriver.NodeID {
			t.Errorf("node ID: got %q, want %q", driver.NodeID, tc.wantDriver.NodeID)
		}
	}
	if tc.wantOtherDriver {
		if !slices.ContainsFunc(updated.Spec.Drivers, func(d storagev1.CSINodeDriver) bool {
			return d.Name == "other.csi.example.com"
		}) {
			t.Errorf("entry of another driver was dropped: %+v", updated.Spec.Drivers)
		}
	}
	if !slices.Equal(updated.Spec.DriverRegistrations, tc.csiNode.Spec.DriverRegistrations) {
		t.Errorf("driver registrations changed: got %+v, want %+v", updated.Spec.DriverRegistrations, tc.csiNode.Spec.DriverRegistrations)
	}

	if len(tc.wantNodeLabels) > 0 {
		node, err := client.CoreV1().Nodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get Node: %v", err)
		}
		for k, v := range tc.wantNodeLabels {
			if node.Labels[k] != v {
				t.Errorf("node label %q: got %q, want %q", k, node.Labels[k], v)
			}
		}
	}
	if writes := writeCount(client.Actions(), "nodes"); writes != tc.wantNodeWrites {
		t.Errorf("Node writes: got %d, want %d", writes, tc.wantNodeWrites)
	}

	events := drainEvents(recorder)
	if tc.wantEvent == "" {
		if len(events) != 0 {
			t.Errorf("unexpected events: %v", events)
		}
		return
	}
	if len(events) != 1 {
		t.Fatalf("events: got %v, want exactly one containing %q", events, tc.wantEvent)
	}
	if !strings.Contains(events[0], tc.wantEvent) {
		t.Errorf("event %q does not contain %q", events[0], tc.wantEvent)
	}
}

func TestSyncNode(t *testing.T) {
	zoneA := topology(map[string]string{zoneKey: "zone-a"})
	zoneAndRegion := topology(map[string]string{zoneKey: "zone-a", regionKey: "region-1"})
	otherNodeVA := testVA(testAttacherName, "pv-other", "other-node")
	otherAttacherVA := testVA("other.csi.example.com", "pv-other", testNodeName)

	testcases := []syncNodeTest{
		{
			name:           "initial registration",
			node:           testNode("node-uid", nil),
			csiNode:        testCSINode("node-uid"),
			resp:           &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10, AccessibleTopology: zoneA},
			wantDriver:     ptr.To(installedDriver([]string{zoneKey}, ptr.To[int32](10))),
			wantNodeLabels: map[string]string{zoneKey: "zone-a"},
			wantNodeWrites: 1,
			wantCalls:      1,
			wantEvent:      `CSINodeRegistered`,
		},
		{
			name:           "initial registration without topology and limit",
			node:           testNode("node-uid", nil),
			csiNode:        testCSINode("node-uid"),
			resp:           &csi.ControllerGetNodeInfoResponse{},
			wantDriver:     ptr.To(installedDriver(nil, nil)),
			wantNodeWrites: 0,
			wantCalls:      1,
			wantEvent:      `allocatableCount=unlimited`,
		},
		{
			name:           "initial registration with existing topology labels",
			node:           testNode("node-uid", map[string]string{zoneKey: "zone-a", "unrelated": "x"}),
			csiNode:        testCSINode("node-uid"),
			resp:           &csi.ControllerGetNodeInfoResponse{AccessibleTopology: zoneA},
			wantDriver:     ptr.To(installedDriver([]string{zoneKey}, nil)),
			wantNodeLabels: map[string]string{zoneKey: "zone-a", "unrelated": "x"},
			wantNodeWrites: 0,
			wantCalls:      1,
			wantEvent:      `CSINodeRegistered`,
		},
		{
			name:           "topology sorted by key",
			node:           testNode("node-uid", nil),
			csiNode:        testCSINode("node-uid"),
			resp:           &csi.ControllerGetNodeInfoResponse{AccessibleTopology: zoneAndRegion},
			wantDriver:     ptr.To(installedDriver([]string{regionKey, zoneKey}, nil)),
			wantNodeLabels: map[string]string{zoneKey: "zone-a", regionKey: "region-1"},
			wantNodeWrites: 1,
			wantCalls:      1,
			wantEvent:      `CSINodeRegistered`,
		},
		{
			name:            "existing entry of another driver is preserved",
			node:            testNode("node-uid", nil),
			csiNode:         testCSINode("node-uid", storagev1.CSINodeDriver{Name: "other.csi.example.com", NodeID: "other"}),
			resp:            &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 3},
			wantDriver:      ptr.To(installedDriver(nil, ptr.To[int32](3))),
			wantOtherDriver: true,
			wantCalls:       1,
			wantEvent:       `CSINodeRegistered`,
		},
		{
			name:       "topology collision with existing Node label",
			node:       testNode("node-uid", map[string]string{zoneKey: "zone-b"}),
			csiNode:    testCSINode("node-uid"),
			resp:       &csi.ControllerGetNodeInfoResponse{AccessibleTopology: zoneA},
			wantErr:    "topology value collision",
			wantCalls:  1,
			wantDriver: nil,
		},
		{
			name:      "CSINode not owned by Node",
			node:      testNode("node-uid", nil),
			csiNode:   func() *storagev1.CSINode { n := testCSINode("other-uid"); return n }(),
			resp:      &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 3},
			wantErr:   "CSINode is not owned by this Node",
			wantCalls: 1,
		},
		{
			name:      "Node object is gone",
			csiNode:   testCSINode("node-uid"),
			resp:      &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 3},
			wantErr:   "get Node",
			wantCalls: 1,
		},
		{
			name:        "ControllerGetNodeInfo fails",
			node:        testNode("node-uid", nil),
			csiNode:     testCSINode("node-uid"),
			providerErr: errors.New("boom"),
			wantErr:     "call ControllerGetNodeInfo: boom",
			wantCalls:   1,
		},
		{
			name:      "ControllerGetNodeInfo times out",
			node:      testNode("node-uid", nil),
			csiNode:   testCSINode("node-uid"),
			block:     true,
			timeout:   time.Millisecond,
			wantErr:   context.DeadlineExceeded.Error(),
			wantCalls: 1,
		},
		{
			name:       "non-CSI volumes are deducted from the limit",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid"),
			objects:    []runtime.Object{testPV("pv-csi", "vol-csi"), testVA(testAttacherName, "pv-csi", testNodeName)},
			resp:       &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10, PublishedVolumes: publishedVolumes("vol-csi", "vol-boot", "vol-manual")},
			wantDriver: ptr.To(installedDriver(nil, ptr.To[int32](8))),
			wantCalls:  1,
			wantEvent:  `allocatableCount=8`,
		},
		{
			name:       "all published volumes are CSI-managed",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid"),
			objects:    []runtime.Object{testPV("pv-csi", "vol-csi"), testVA(testAttacherName, "pv-csi", testNodeName)},
			resp:       &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10, PublishedVolumes: publishedVolumes("vol-csi")},
			wantDriver: ptr.To(installedDriver(nil, ptr.To[int32](10))),
			wantCalls:  1,
			wantEvent:  `allocatableCount=10`,
		},
		{
			name:       "no published volumes, no deduction",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid"),
			objects:    []runtime.Object{testPV("pv-csi", "vol-csi"), testVA(testAttacherName, "pv-csi", testNodeName)},
			resp:       &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10},
			wantDriver: ptr.To(installedDriver(nil, ptr.To[int32](10))),
			wantCalls:  1,
			wantEvent:  `allocatableCount=10`,
		},
		{
			name:       "limit is not negative",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid"),
			resp:       &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 1, PublishedVolumes: publishedVolumes("vol-boot", "vol-manual")},
			wantDriver: ptr.To(installedDriver(nil, ptr.To[int32](0))),
			wantCalls:  1,
			wantEvent:  `allocatableCount=0`,
		},
		{
			name:       "VAs of other nodes and other attachers are ignored",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid"),
			objects:    []runtime.Object{testPV("pv-other", "vol-other"), otherNodeVA, otherAttacherVA},
			resp:       &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10, PublishedVolumes: publishedVolumes("vol-other")},
			wantDriver: ptr.To(installedDriver(nil, ptr.To[int32](9))),
			wantCalls:  1,
			wantEvent:  `allocatableCount=9`,
		},
		{
			name:      "missing PV of a VA fails the whole update",
			node:      testNode("node-uid", nil),
			csiNode:   testCSINode("node-uid"),
			objects:   []runtime.Object{testVA(testAttacherName, "pv-missing", testNodeName)},
			resp:      &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10, PublishedVolumes: publishedVolumes("vol")},
			wantErr:   `resolve source for VA "pv-missing-node1"`,
			wantCalls: 1,
		},
		{
			name:       "VA with inline volume spec",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid"),
			objects:    []runtime.Object{inlineVA()},
			resp:       &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10, PublishedVolumes: publishedVolumes(testVolumeHandle)},
			wantDriver: ptr.To(installedDriver(nil, ptr.To[int32](10))),
			wantCalls:  1,
			wantEvent:  `allocatableCount=10`,
		},
		{
			name:       "refresh of an installed entry",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid", installedDriver([]string{zoneKey}, ptr.To[int32](10))),
			resp:       &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 8, AccessibleTopology: zoneA},
			refresh:    true,
			wantDriver: ptr.To(installedDriver([]string{zoneKey}, ptr.To[int32](8))),
			wantCalls:  1,
			wantEvent:  `CSINodeInfoUpdated`,
		},
		{
			name:       "refresh keeps topology keys and Node labels",
			node:       testNode("node-uid", map[string]string{zoneKey: "zone-b"}),
			csiNode:    testCSINode("node-uid", installedDriver([]string{zoneKey}, nil)),
			resp:       &csi.ControllerGetNodeInfoResponse{AccessibleTopology: zoneA},
			refresh:    true,
			wantDriver: ptr.To(installedDriver([]string{zoneKey}, nil)),
			wantCalls:  1,
			wantEvent:  `CSINodeInfoUpdated`,
		},
		{
			// We did not create this entry, so we must not remove what we do not own.
			name:       "refresh keeps an allocatable owned by another manager",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid", installedDriver(nil, ptr.To[int32](10))),
			resp:       &csi.ControllerGetNodeInfoResponse{},
			refresh:    true,
			wantDriver: ptr.To(installedDriver(nil, ptr.To[int32](10))),
			wantCalls:  1,
			wantEvent:  `CSINodeInfoUpdated`,
		},
		{
			name:       "installed entry without refresh is not queried",
			node:       testNode("node-uid", nil),
			csiNode:    testCSINode("node-uid", installedDriver(nil, ptr.To[int32](10))),
			resp:       &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 8},
			wantDriver: ptr.To(installedDriver(nil, ptr.To[int32](10))),
			wantCalls:  0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			runSyncNodeTest(t, tc)
		})
	}
}

// Recording a volume while ControllerGetNodeInfo is in flight makes the volume
// count as CSI-managed, even though no VolumeAttachment exists for it yet.
func TestSyncNodeRecordsVolumesDuringQuery(t *testing.T) {
	h, client, recorder, provider := newNodeTestHandler(t, nil,
		testNode("node-uid", nil),
		testCSINode("node-uid"),
		testPV("pv-csi", "vol-csi"),
		testVA(testAttacherName, "pv-csi", testNodeName),
	)
	provider.resp = &csi.ControllerGetNodeInfoResponse{
		MaxVolumesPerNode: 10,
		// vol-attaching has no VolumeAttachment yet: it is attached by a
		// concurrent ControllerPublishVolume.
		PublishedVolumes: publishedVolumes("vol-csi", "vol-attaching", "vol-boot"),
	}
	provider.onCall = func(nodeID string) {
		if nodeID != testNodeID {
			t.Errorf("nodeID: got %q, want %q", nodeID, testNodeID)
		}
		h.RecordProcessing(nodeID, "vol-attaching")
	}

	if err := h.SyncNode(t.Context(), testCSINode("node-uid"), testNodeID, false); err != nil {
		t.Fatalf("SyncNode: %v", err)
	}

	updated, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSINode: %v", err)
	}
	if got := ptr.Deref(updated.Spec.Drivers[0].Allocatable.Count, -1); got != 9 {
		t.Errorf("allocatable count: got %d, want 9 (only vol-boot is non-CSI)", got)
	}
	if events := drainEvents(recorder); len(events) != 1 {
		t.Errorf("events: got %v, want 1", events)
	}

	// Outside of a ControllerGetNodeInfo call, recording is a no-op and must not panic.
	count := 0
	h.processingVolumes.Range(func(any, any) bool {
		count++
		return true
	})
	if count != 0 {
		t.Errorf("processing volumes were not cleaned up: %d left", count)
	}
}

// A driver that stops reporting a limit must have the limit we published removed.
func TestSyncNodeRefreshRemovesOwnedLimit(t *testing.T) {
	h, client, _, provider := newNodeTestHandler(t, nil, testNode("node-uid", nil), testCSINode("node-uid"))
	provider.resp = &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10}
	if err := h.SyncNode(t.Context(), testCSINode("node-uid"), testNodeID, false); err != nil {
		t.Fatalf("register: %v", err)
	}
	csiNode, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSINode: %v", err)
	}

	provider.resp = &csi.ControllerGetNodeInfoResponse{}
	if err := h.SyncNode(t.Context(), csiNode, testNodeID, true); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	updated, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSINode: %v", err)
	}
	if updated.Spec.Drivers[0].Allocatable != nil {
		t.Errorf("allocatable: got %+v, want unset", updated.Spec.Drivers[0].Allocatable)
	}
}

func TestSyncNodeMigratedVolume(t *testing.T) {
	const gceNodeID = "projects/test-project/zones/testZone/instances/test-node"
	const repairedHandle = "projects/test-project/zones/testZone/disks/testpd"

	h, client, _, provider := newNodeTestHandler(t, csitranslator.New(),
		testNode("node-uid", nil),
		testCSINode("node-uid"),
		gcePDPV(),
		testVA(testAttacherName, testPVName, testNodeName),
	)
	provider.resp = &csi.ControllerGetNodeInfoResponse{
		MaxVolumesPerNode: 10,
		PublishedVolumes:  publishedVolumes(repairedHandle, "vol-boot"),
	}

	if err := h.SyncNode(t.Context(), testCSINode("node-uid"), gceNodeID, false); err != nil {
		t.Fatalf("SyncNode: %v", err)
	}

	updated, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSINode: %v", err)
	}
	if got := ptr.Deref(updated.Spec.Drivers[0].Allocatable.Count, -1); got != 9 {
		t.Errorf("allocatable count: got %d, want 9 (the migrated volume must not be deducted)", got)
	}
}

// The first refresh of an entry created by kubelet must claim the fields so that
// later updates can also remove them.
func TestSyncNodeRefreshClaimsOwnership(t *testing.T) {
	csiNode := testCSINode("node-uid", installedDriver([]string{zoneKey}, ptr.To[int32](10)))
	h, client, recorder, provider := newNodeTestHandler(t, nil, testNode("node-uid", nil), csiNode)
	provider.resp = &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10, AccessibleTopology: topology(map[string]string{zoneKey: "zone-a"})}

	snapshot := func() *storagev1.CSINode {
		t.Helper()
		snapshot, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get CSINode: %v", err)
		}
		return snapshot
	}

	// kubelet wrote the entry, so the fake client has no managed fields for us yet.
	client.ClearActions()
	if err := h.SyncNode(t.Context(), snapshot(), testNodeID, true); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if writes := writeCount(client.Actions(), "csinodes"); writes != 1 {
		t.Errorf("CSINode writes: got %d, want 1", writes)
	}
	if events := drainEvents(recorder); len(events) != 1 {
		t.Errorf("events: got %v, want 1", events)
	}

	// We own the fields now, an identical refresh must not write.
	client.ClearActions()
	if err := h.SyncNode(t.Context(), snapshot(), testNodeID, true); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if writes := writeCount(client.Actions(), "csinodes"); writes != 0 {
		t.Errorf("CSINode writes: got %d, want 0", writes)
	}
	if events := drainEvents(recorder); len(events) != 0 {
		t.Errorf("unexpected events: %v", events)
	}
}

// A conflicting write is dropped: the next CSINode event retries with a fresh
// snapshot, which may have unregistered the driver in the meantime.
func TestSyncNodeConflictIsIgnored(t *testing.T) {
	csiNode := testCSINode("node-uid")
	h, client, recorder, provider := newNodeTestHandler(t, nil, testNode("node-uid", nil), csiNode)
	provider.resp = &csi.ControllerGetNodeInfoResponse{MaxVolumesPerNode: 10}
	client.PrependReactor("patch", "csinodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrs.NewConflict(schema.GroupResource{Group: "storage.k8s.io", Resource: "csinodes"}, testNodeName, errors.New("conflict"))
	})

	if err := h.SyncNode(t.Context(), csiNode, testNodeID, false); err != nil {
		t.Fatalf("SyncNode: %v", err)
	}
	if events := drainEvents(recorder); len(events) != 0 {
		t.Errorf("unexpected events: %v", events)
	}
}

func TestSyncNodeNodePatchFails(t *testing.T) {
	csiNode := testCSINode("node-uid")
	h, client, _, provider := newNodeTestHandler(t, nil, testNode("node-uid", nil), csiNode)
	provider.resp = &csi.ControllerGetNodeInfoResponse{AccessibleTopology: topology(map[string]string{zoneKey: "zone-a"})}
	client.PrependReactor("patch", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("apply failed")
	})

	err := h.SyncNode(t.Context(), csiNode, testNodeID, false)
	if err == nil || !strings.Contains(err.Error(), "apply Node labels") {
		t.Fatalf("error: got %v, want an apply Node labels error", err)
	}
	if writes := writeCount(client.Actions(), "csinodes"); writes != 0 {
		t.Errorf("CSINode writes: got %d, want 0", writes)
	}
}

func TestNodeFailingHandler(t *testing.T) {
	handler := NodeFailingHandler{Err: errors.New("driver does not advertise GET_NODE_INFO")}
	if !handler.HasSynced() {
		t.Error("NodeFailingHandler must be usable immediately")
	}
	err := handler.SyncNode(t.Context(), testCSINode("node-uid"), testNodeID, false)
	if !errors.Is(err, handler.Err) {
		t.Errorf("SyncNode error: got %v, want %v", err, handler.Err)
	}
}

func TestNodeCSIHandlerHasSynced(t *testing.T) {
	h, _, _, _ := newNodeTestHandler(t, nil)
	if !h.HasSynced() {
		t.Error("handler is not synced although the informers are")
	}
}
