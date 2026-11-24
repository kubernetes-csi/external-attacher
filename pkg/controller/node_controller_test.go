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
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	csitranslator "k8s.io/csi-translation-lib"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/ktesting"
	_ "k8s.io/klog/v2/ktesting/init"
	"k8s.io/utils/ptr"
)

const (
	updatePeriod   = 60 * time.Second
	firstUpdate    = updatePeriod / 2            // randN returns n/2 in tests, see newNodeController
	updateJitter   = updatePeriod / 25 / 2       // half of the maximum jitter
	updateInterval = updatePeriod + updateJitter // between two periodic updates
)

type nodeSyncCall struct {
	nodeName string
	nodeID   string
	refresh  bool
	at       time.Time
}

// testNodeInfoHandler records the ControllerGetNodeInfo queries the controller
// triggers. Like the real handler, it does not query an already installed driver
// unless a refresh is requested.
type testNodeInfoHandler struct {
	mu    sync.Mutex
	err   error
	calls []nodeSyncCall
}

func (h *testNodeInfoHandler) HasSyncedCheckers() []cache.DoneChecker { return nil }

func (h *testNodeInfoHandler) SyncNode(_ context.Context, csiNode *storagev1.CSINode, nodeID string, refresh bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	installed := slices.ContainsFunc(csiNode.Spec.Drivers, func(d storagev1.CSINodeDriver) bool {
		return d.Name == testAttacherName && d.NodeID == nodeID
	})
	if installed && !refresh {
		return h.err
	}
	h.calls = append(h.calls, nodeSyncCall{nodeName: csiNode.Name, nodeID: nodeID, refresh: refresh, at: time.Now()})
	return h.err
}

// take returns the calls recorded since the last take.
func (h *testNodeInfoHandler) take() []nodeSyncCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	calls := h.calls
	h.calls = nil
	return calls
}

func (h *testNodeInfoHandler) snapshot() []nodeSyncCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.calls)
}

func (h *testNodeInfoHandler) setErr(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.err = err
}

type nodeControllerFixture struct {
	ctrl     *NodeController
	client   *fake.Clientset
	recorder *record.FakeRecorder
	handler  *testNodeInfoHandler
}

func newNodeController(logger klog.Logger, informerFactory informers.SharedInformerFactory, recorder *record.FakeRecorder, handler NodeInfoHandler) *NodeController {
	ctrl := NewNodeController(
		logger,
		testAttacherName,
		handler,
		recorder,
		informerFactory.Storage().V1().CSINodes(),
		informerFactory.Storage().V1().CSIDrivers(),
		workqueue.DefaultTypedControllerRateLimiter[string](),
	)
	ctrl.randN = func(n time.Duration) time.Duration { return n / 2 }
	return ctrl
}

func newTestNodeControllerFixture(t *testing.T, handlerErr error, objs ...runtime.Object) *nodeControllerFixture {
	t.Helper()
	client := fake.NewClientset(objs...)
	informerFactory := informers.NewSharedInformerFactory(client, 0)
	recorder := record.NewFakeRecorder(100)
	handler := &testNodeInfoHandler{err: handlerErr}
	logger := ktesting.NewLogger(t, ktesting.DefaultConfig)
	ctx := klog.NewContext(t.Context(), logger)

	ctrl := newNodeController(logger, informerFactory, recorder, handler)

	done := make(chan struct{})
	informerFactory.StartWithContext(ctx)
	go func() {
		ctrl.Run(ctx, 1)
		close(done)
	}()
	t.Cleanup(func() { <-done })
	return &nodeControllerFixture{ctrl: ctrl, client: client, recorder: recorder, handler: handler}
}

func (f *nodeControllerFixture) stateSize() int {
	f.ctrl.stateMu.Lock()
	defer f.ctrl.stateMu.Unlock()
	return len(f.ctrl.state)
}

// installedNodes returns num CSINodes with our registration and driver entry.
func installedNodes(num int) []runtime.Object {
	objs := make([]runtime.Object, 0, num)
	for i := range num {
		csiNode := testCSINode("node-uid", installedDriver(nil, new(int32(10))))
		csiNode.Name = fmt.Sprintf("node-%d", i)
		objs = append(objs, csiNode)
	}
	return objs
}

func testCSIDriver(period *int64) *storagev1.CSIDriver {
	return &storagev1.CSIDriver{
		ObjectMeta: metav1.ObjectMeta{Name: testAttacherName},
		Spec:       storagev1.CSIDriverSpec{NodeAllocatableUpdatePeriodSeconds: period},
	}
}

func mustUpdateCSINode(t *testing.T, client *fake.Clientset, mutate func(*storagev1.CSINode)) {
	t.Helper()
	csiNode, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSINode: %v", err)
	}
	mutate(csiNode)
	if _, err := client.StorageV1().CSINodes().Update(t.Context(), csiNode, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update CSINode: %v", err)
	}
}

func mustUpdateCSIDriver(t *testing.T, client *fake.Clientset, period *int64) {
	t.Helper()
	driver, err := client.StorageV1().CSIDrivers().Get(t.Context(), testAttacherName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get CSIDriver: %v", err)
	}
	driver.Spec.NodeAllocatableUpdatePeriodSeconds = period
	if _, err := client.StorageV1().CSIDrivers().Update(t.Context(), driver, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update CSIDriver: %v", err)
	}
}

type schedulingScenario struct {
	name    string
	objects []runtime.Object
	// handlerErr makes the initial SyncNode fail
	handlerErr error
	run        func(t *testing.T, f *nodeControllerFixture)
}

func TestNodeControllerScheduling(t *testing.T) {
	periodSeconds := int64(updatePeriod / time.Second)
	pending := func() *storagev1.CSINode { return testCSINode("node-uid") }
	installed := func() *storagev1.CSINode {
		return testCSINode("node-uid", installedDriver(nil, new(int32(10))))
	}

	scenarios := []schedulingScenario{
		{
			name:    "pending registration is installed immediately, then refreshed periodically",
			objects: []runtime.Object{pending(), testCSIDriver(&periodSeconds)},
			run: func(t *testing.T, f *nodeControllerFixture) {
				calls := f.handler.take()
				if len(calls) != 1 {
					t.Fatalf("calls: got %v, want the registration only", calls)
				}
				if calls[0].nodeID != testNodeID {
					t.Errorf("nodeID: got %q, want %q", calls[0].nodeID, testNodeID)
				}
				installedAt := calls[0].at

				// A node installed by the controller gets its first periodic update
				// at a random offset within the period, half of it with the pinned randN.
				synctest.Sleep(firstUpdate - time.Millisecond)
				if calls := f.handler.take(); len(calls) != 0 {
					t.Fatalf("updated before the scheduled time: %v", calls)
				}

				synctest.Sleep(time.Millisecond)
				calls = f.handler.take()
				if len(calls) != 1 || !calls[0].refresh {
					t.Fatalf("calls after the first period: got %v, want one refresh", calls)
				}
				if got := calls[0].at.Sub(installedAt); got != firstUpdate {
					t.Errorf("periodic update %v after the registration, want %v", got, firstUpdate)
				}

				// Later updates keep the period, plus half the maximum jitter.
				synctest.Sleep(updateInterval)
				calls = f.handler.take()
				if len(calls) != 1 {
					t.Fatalf("calls after the second period: got %v, want 1", calls)
				}
			},
		},
		{
			name:    "installed at startup without periodic updates is not queried",
			objects: []runtime.Object{installed()},
			run: func(t *testing.T, f *nodeControllerFixture) {
				synctest.Sleep(2 * time.Hour)
				if calls := f.handler.snapshot(); len(calls) != 0 {
					t.Fatalf("unexpected calls: %v", calls)
				}
				if size := f.stateSize(); size != 0 {
					t.Errorf("tracked nodes: got %d, want 0", size)
				}
			},
		},
		{
			// A restart must not refresh all installed nodes at once: they are
			// spread evenly over the period and keep that slot.
			name:    "installed nodes are spread evenly over the period",
			objects: append(installedNodes(3), testCSIDriver(&periodSeconds)),
			run: func(t *testing.T, f *nodeControllerFixture) {
				start := time.Now()
				// Only the node with the first slot is due immediately.
				if calls := f.handler.snapshot(); len(calls) != 1 {
					t.Fatalf("calls after start: got %d, want 1", len(calls))
				}

				for i := 1; i < 3; i++ {
					// The slots are one third of the period apart.
					slot := start.Add(time.Duration(i) * updatePeriod / 3)
					synctest.Sleep(time.Until(slot.Add(-time.Second)))
					if calls := f.handler.snapshot(); len(calls) != i {
						t.Fatalf("calls before slot %d: got %d, want %d", i, len(calls), i)
					}
					synctest.Sleep(time.Second)
					calls := f.handler.snapshot()
					if len(calls) != i+1 {
						t.Fatalf("calls after slot %d: got %d, want %d", i, len(calls), i+1)
					}
					if !calls[i].refresh {
						t.Errorf("scheduled update %d was not a refresh: %+v", i, calls[i])
					}
				}

				// The first node keeps its slot: its next update is one period later.
				synctest.Sleep(time.Until(start.Add(updatePeriod + updatePeriod/10)))
				if calls := f.handler.snapshot(); len(calls) != 4 {
					t.Fatalf("calls after one period: got %d, want 4", len(calls))
				}
			},
		},
		{
			name:    "re-registration with a preserved entry does not force a lookup, kubelet kept it fresh",
			objects: []runtime.Object{installed(), testCSIDriver(&periodSeconds)},
			run: func(t *testing.T, f *nodeControllerFixture) {
				if calls := f.handler.take(); len(calls) != 1 {
					t.Fatalf("calls after start: got %d, want 1", len(calls))
				}

				// kubelet removes the registration, but keeps Spec.Drivers when the
				// driver registers again after a rollout.
				mustUpdateCSINode(t, f.client, func(n *storagev1.CSINode) { n.Spec.DriverRegistrations = nil })
				synctest.Wait()
				// The already scheduled update still runs, but must not query the
				// driver anymore. Afterwards the node is not tracked at all.
				synctest.Sleep(updateInterval)
				if calls := f.handler.take(); len(calls) != 0 {
					t.Fatalf("calls while unregistered: got %v", calls)
				}
				if size := f.stateSize(); size != 0 {
					t.Errorf("tracked nodes after unregistration: got %d, want 0", size)
				}

				// Re-registration with a preserved entry: kubelet refreshed that entry
				// while the driver was in the node-side flow, so there is nothing to
				// refresh now. The first controller-side update gets a random offset,
				// which keeps the expected interval between two updates at one period.
				mustUpdateCSINode(t, f.client, func(n *storagev1.CSINode) {
					n.Spec.DriverRegistrations = []storagev1.CSINodeDriverRegistration{{Name: testAttacherName, NodeID: testNodeID}}
				})
				synctest.Wait()
				if calls := f.handler.take(); len(calls) != 0 {
					t.Fatalf("re-registration forced a lookup: %v", calls)
				}
				synctest.Sleep(firstUpdate - time.Millisecond)
				if calls := f.handler.take(); len(calls) != 0 {
					t.Fatalf("updated before the scheduled time: %v", calls)
				}
				synctest.Sleep(time.Millisecond)
				if calls := f.handler.take(); len(calls) != 1 || !calls[0].refresh {
					t.Fatalf("calls at the scheduled time: got %v, want one refresh", calls)
				}

				// The same cycle, but the registration comes back while the node is
				// still tracked (the state is only dropped when an update is due).
				mustUpdateCSINode(t, f.client, func(n *storagev1.CSINode) { n.Spec.DriverRegistrations = nil })
				synctest.Wait()
				mustUpdateCSINode(t, f.client, func(n *storagev1.CSINode) {
					n.Spec.DriverRegistrations = []storagev1.CSINodeDriverRegistration{{Name: testAttacherName, NodeID: testNodeID}}
				})
				synctest.Wait()
				if calls := f.handler.take(); len(calls) != 0 {
					t.Fatalf("re-registration with an intact schedule forced a lookup: %v", calls)
				}
			},
		},
		{
			name:    "re-registration without a preserved entry registers again",
			objects: []runtime.Object{installed(), testCSIDriver(&periodSeconds)},
			run: func(t *testing.T, f *nodeControllerFixture) {
				if calls := f.handler.take(); len(calls) != 1 {
					t.Fatalf("calls after start: got %d, want 1", len(calls))
				}

				// kubelet unregisters: both entries are removed in one update.
				mustUpdateCSINode(t, f.client, func(n *storagev1.CSINode) {
					n.Spec.DriverRegistrations = nil
					n.Spec.Drivers = nil
				})
				synctest.Wait()
				mustUpdateCSINode(t, f.client, func(n *storagev1.CSINode) {
					n.Spec.DriverRegistrations = []storagev1.CSINodeDriverRegistration{{Name: testAttacherName, NodeID: testNodeID}}
				})
				synctest.Wait()
				calls := f.handler.take()
				if len(calls) != 1 || calls[0].nodeID != testNodeID {
					t.Fatalf("calls after re-registration: got %v, want the registration", calls)
				}
			},
		},
		{
			name:    "RequestUpdate triggers an immediate refresh",
			objects: []runtime.Object{installed(), testCSIDriver(&periodSeconds)},
			run: func(t *testing.T, f *nodeControllerFixture) {
				f.handler.take()
				if err := f.ctrl.RequestUpdate(t.Context(), testNodeName); err != nil {
					t.Fatalf("RequestUpdate: %v", err)
				}
				synctest.Wait()
				calls := f.handler.take()
				if len(calls) != 1 || !calls[0].refresh {
					t.Fatalf("calls after RequestUpdate: got %v, want one refresh", calls)
				}
				// The periodic update keeps its schedule.
				synctest.Sleep(updateInterval - time.Millisecond)
				if calls := f.handler.take(); len(calls) != 0 {
					t.Fatalf("periodic update moved earlier: %v", calls)
				}
				synctest.Sleep(time.Millisecond)
				if calls := f.handler.take(); len(calls) != 1 || !calls[0].refresh {
					t.Fatalf("calls at the scheduled time: got %v, want one refresh", calls)
				}
			},
		},
		{
			name:    "RequestUpdate without periodic updates is a no-op",
			objects: []runtime.Object{installed()},
			run: func(t *testing.T, f *nodeControllerFixture) {
				if err := f.ctrl.RequestUpdate(t.Context(), testNodeName); err != nil {
					t.Fatalf("RequestUpdate: %v", err)
				}
				synctest.Wait()
				if calls := f.handler.snapshot(); len(calls) != 0 {
					t.Fatalf("unexpected calls: %v", calls)
				}
			},
		},
		{
			name:    "RequestUpdate gives up when its context expires",
			objects: []runtime.Object{installed(), testCSIDriver(&periodSeconds)},
			run: func(t *testing.T, f *nodeControllerFixture) {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				if err := f.ctrl.RequestUpdate(ctx, testNodeName); !errors.Is(err, context.Canceled) {
					t.Errorf("RequestUpdate: got %v, want %v", err, context.Canceled)
				}
			},
		},
		{
			name:       "failures are retried and reported as events",
			objects:    []runtime.Object{pending()},
			handlerErr: errors.New("boom"),
			run: func(t *testing.T, f *nodeControllerFixture) {
				calls := f.handler.snapshot()
				if len(calls) == 0 {
					t.Fatal("driver was not processed")
				}
				events := drainEvents(f.recorder)
				if len(events) == 0 {
					t.Fatal("no event reported")
				}
				for _, want := range []string{"Warning", "CSINodeInfoUpdateFailed", testAttacherName, "boom"} {
					if !strings.Contains(events[0], want) {
						t.Errorf("event %q does not contain %q", events[0], want)
					}
				}

				// The rate limiter retries with a growing delay, it never gives up.
				f.handler.setErr(nil)
				synctest.Sleep(time.Minute)
				before := len(f.handler.snapshot())
				synctest.Sleep(10 * time.Minute)
				if calls := f.handler.snapshot(); len(calls) != before {
					t.Errorf("retried after success: got %d calls, want %d", len(calls), before)
				}
				if size := f.stateSize(); size != 0 {
					t.Errorf("tracked nodes: got %d, want 0", size)
				}
			},
		},
		{
			name:    "period change reschedules the nodes",
			objects: []runtime.Object{installed(), testCSIDriver(&periodSeconds)},
			run: func(t *testing.T, f *nodeControllerFixture) {
				if calls := f.handler.take(); len(calls) != 1 {
					t.Fatalf("calls after start: got %d, want 1", len(calls))
				}

				const (
					shortPeriod = 5 * time.Second
					shortJitter = shortPeriod / 25 / 2 // half of the maximum jitter
				)
				mustUpdateCSIDriver(t, f.client, new(int64(shortPeriod/time.Second)))
				synctest.Wait()
				// The update was scheduled for the old period, it is moved into the
				// new one. The only node gets the first slot, so it is due right away.
				if calls := f.handler.take(); len(calls) != 1 || !calls[0].refresh {
					t.Fatalf("calls after the period change: got %v, want one refresh", calls)
				}
				// The following updates use the new period.
				synctest.Sleep(shortPeriod + shortJitter - time.Millisecond)
				if calls := f.handler.take(); len(calls) != 0 {
					t.Fatalf("updated before the short period elapsed: %v", calls)
				}
				synctest.Sleep(time.Millisecond)
				if calls := f.handler.take(); len(calls) != 1 || !calls[0].refresh {
					t.Fatalf("calls with the short period: got %v, want one refresh", calls)
				}
			},
		},
		{
			name:    "deleting the CSIDriver disables periodic updates",
			objects: []runtime.Object{installed(), testCSIDriver(&periodSeconds)},
			run: func(t *testing.T, f *nodeControllerFixture) {
				if calls := f.handler.take(); len(calls) != 1 {
					t.Fatalf("calls after start: got %d, want 1", len(calls))
				}

				if err := f.client.StorageV1().CSIDrivers().Delete(t.Context(), testAttacherName, metav1.DeleteOptions{}); err != nil {
					t.Fatalf("delete CSIDriver: %v", err)
				}
				synctest.Wait()
				// The pending update still fires, but with periodic updates disabled
				// it does not query the driver and drops the node.
				synctest.Sleep(updateInterval)
				if calls := f.handler.take(); len(calls) != 0 {
					t.Fatalf("refresh after periodic updates were disabled: %v", calls)
				}
				if size := f.stateSize(); size != 0 {
					t.Errorf("tracked nodes: got %d, want 0", size)
				}
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newTestNodeControllerFixture(t, sc.handlerErr, sc.objects...)
				// The controller's cache sync is channel-based, so this also waits for
				// the initial events to be processed.
				synctest.Wait()
				sc.run(t, f)
			})
		})
	}
}

func TestProcessNode(t *testing.T) {
	testcases := []struct {
		name         string
		csiNode      *storagev1.CSINode
		handlerErr   error
		wantManaged  bool
		wantCalls    int
		wantEventSub string
	}{
		{
			name:        "success",
			csiNode:     testCSINode("node-uid"),
			wantManaged: true,
			wantCalls:   1,
		},
		{
			name:         "failure is reported as an event",
			csiNode:      testCSINode("node-uid"),
			handlerErr:   errors.New("get node info: boom"),
			wantManaged:  true,
			wantCalls:    1,
			wantEventSub: "get node info: boom",
		},
		{
			name: "not registered",
			csiNode: func() *storagev1.CSINode {
				n := testCSINode("node-uid")
				n.Spec.DriverRegistrations = nil
				return n
			}(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &testNodeInfoHandler{err: tc.handlerErr}
			client := fake.NewClientset(tc.csiNode)
			factory := informers.NewSharedInformerFactory(client, 0)
			csiNodeInformer := factory.Storage().V1().CSINodes()
			if err := csiNodeInformer.Informer().GetStore().Add(tc.csiNode); err != nil {
				t.Fatalf("add CSINode to store: %v", err)
			}
			recorder := record.NewFakeRecorder(100)
			logger, _ := ktesting.NewTestContext(t)
			ctrl := NewNodeController(logger, testAttacherName, handler, recorder,
				csiNodeInformer, factory.Storage().V1().CSIDrivers(),
				workqueue.DefaultTypedControllerRateLimiter[string]())

			managed, err := ctrl.processNode(testContext(t), testNodeName, false)
			if tc.handlerErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.handlerErr != nil && err == nil {
				t.Fatal("expected error")
			}
			if managed != tc.wantManaged {
				t.Errorf("managed: got %v, want %v", managed, tc.wantManaged)
			}
			if calls := handler.snapshot(); len(calls) != tc.wantCalls {
				t.Errorf("calls: got %v, want %d", calls, tc.wantCalls)
			}
			events := drainEvents(recorder)
			if tc.wantEventSub == "" {
				if len(events) != 0 {
					t.Errorf("unexpected events: %v", events)
				}
				return
			}
			if len(events) != 1 || !strings.Contains(events[0], tc.wantEventSub) {
				t.Errorf("events: got %v, want one containing %q", events, tc.wantEventSub)
			}
		})
	}
}

func TestProcessNodeDeleted(t *testing.T) {
	f := newTestNodeControllerFixture(t, nil)
	managed, err := f.ctrl.processNode(testContext(t), "missing-node", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed {
		t.Error("deleted node must not be managed")
	}
	if calls := f.handler.snapshot(); len(calls) != 0 {
		t.Errorf("unexpected calls: %v", calls)
	}
}

// TestNodeControllerIntegration runs the real handler against a fake API server:
// registration of a pending driver, periodic capacity refresh, disabling periodic
// updates and a controller restart with already installed drivers.
func TestNodeControllerIntegration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const periodSeconds = int64(updatePeriod / time.Second)
		logger := ktesting.NewLogger(t, ktesting.DefaultConfig)
		ctx := klog.NewContext(t.Context(), logger)
		node := testNode("node-uid", nil)
		pv := testPV("pv-csi", "vol-csi")
		attachment := testVA(testAttacherName, "pv-csi", testNodeName)
		provider := &testNodeInfoProvider{resp: &csi.ControllerGetNodeInfoResponse{
			MaxVolumesPerNode:  25,
			AccessibleTopology: topology(map[string]string{zoneKey: "zone-a"}),
			PublishedVolumes:   publishedVolumes("vol-csi", "vol-boot"),
		}}
		client := fake.NewClientset(node, testCSINode("node-uid"), pv, attachment, testCSIDriver(new(periodSeconds)))

		var recorder *record.FakeRecorder
		start := func() func() {
			ctx := ctx
			ctx, cancel := context.WithCancel(ctx)
			factory := informers.NewSharedInformerFactory(client, 0)
			recorder = record.NewFakeRecorder(100)
			h := NewNodeCSIHandler(
				testAttacherName,
				client,
				provider,
				time.Minute,
				recorder,
				factory.Storage().V1().VolumeAttachments().TypedInformer(),
				factory.Core().V1().PersistentVolumes(),
				csitranslator.New(),
			)
			ctrl := newNodeController(logger, factory, recorder, h)
			factory.StartWithContext(ctx)
			done := make(chan struct{})
			go func() {
				ctrl.Run(ctx, 1)
				close(done)
			}()
			synctest.Wait()
			// Idempotent: cancel is, and receiving from closed done returns immediately.
			stop := func() {
				cancel()
				<-done
			}
			t.Cleanup(stop)
			return stop
		}

		getDriver := func(t *testing.T) *storagev1.CSINodeDriver {
			t.Helper()
			csiNode, err := client.StorageV1().CSINodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("get CSINode: %v", err)
			}
			for i := range csiNode.Spec.Drivers {
				if csiNode.Spec.Drivers[i].Name == testAttacherName {
					return &csiNode.Spec.Drivers[i]
				}
			}
			return nil
		}

		stop := start()

		// Initial registration: topology, non-CSI volume deducted, event emitted.
		driver := getDriver(t)
		if driver == nil {
			t.Fatalf("driver was not registered")
		}
		if driver.NodeID != testNodeID {
			t.Errorf("nodeID: got %q, want %q", driver.NodeID, testNodeID)
		}
		if want := []string{zoneKey}; !slices.Equal(driver.TopologyKeys, want) {
			t.Errorf("topology keys: got %v, want %v", driver.TopologyKeys, want)
		}
		if got := ptr.Deref(driver.Allocatable.Count, -1); got != 24 {
			t.Errorf("allocatable count: got %d, want 24", got)
		}
		nodeObj, err := client.CoreV1().Nodes().Get(t.Context(), testNodeName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get Node: %v", err)
		}
		if got := nodeObj.Labels[zoneKey]; got != "zone-a" {
			t.Errorf("node label: got %q, want %q", got, "zone-a")
		}
		if events := drainEvents(recorder); len(events) != 1 || !strings.Contains(events[0], "CSINodeRegistered") {
			t.Errorf("events: got %v, want one CSINodeRegistered", events)
		}

		// Periodic refresh picks up a newly attached non-CSI volume.
		provider.set(&csi.ControllerGetNodeInfoResponse{
			MaxVolumesPerNode:  25,
			AccessibleTopology: topology(map[string]string{zoneKey: "zone-a"}),
			PublishedVolumes:   publishedVolumes("vol-csi", "vol-boot", "vol-manual"),
		}, nil)
		synctest.Sleep(firstUpdate)
		if got := ptr.Deref(getDriver(t).Allocatable.Count, -1); got != 23 {
			t.Errorf("allocatable count after periodic refresh: got %d, want 23", got)
		}
		if events := drainEvents(recorder); len(events) != 1 || !strings.Contains(events[0], "CSINodeInfoUpdated") {
			t.Errorf("events: got %v, want one CSINodeInfoUpdated", events)
		}

		// Disabling periodic updates stops the ControllerGetNodeInfo calls but
		// keeps the published information.
		if err := client.StorageV1().CSIDrivers().Delete(t.Context(), testAttacherName, metav1.DeleteOptions{}); err != nil {
			t.Fatalf("delete CSIDriver: %v", err)
		}
		synctest.Wait()
		calls := provider.callCount()
		synctest.Sleep(updateInterval)
		if got := provider.callCount(); got != calls {
			t.Errorf("ControllerGetNodeInfo calls after disabling periodic updates: got %d, want %d", got, calls)
		}
		if got := ptr.Deref(getDriver(t).Allocatable.Count, -1); got != 23 {
			t.Errorf("allocatable count was not preserved: got %d, want 23", got)
		}

		// Restarting the controller must not force a lookup of the installed driver.
		stop()
		synctest.Wait()

		start()
		synctest.Sleep(2 * updatePeriod)
		if got := provider.callCount(); got != calls {
			t.Errorf("ControllerGetNodeInfo calls after restart: got %d, want %d", got, calls)
		}
		if got := ptr.Deref(getDriver(t).Allocatable.Count, -1); got != 23 {
			t.Errorf("allocatable count after restart: got %d, want 23", got)
		}

		// Re-enabling periodic updates refreshes without re-registering.
		provider.set(&csi.ControllerGetNodeInfoResponse{
			MaxVolumesPerNode:  25,
			AccessibleTopology: topology(map[string]string{zoneKey: "zone-a"}),
			PublishedVolumes:   publishedVolumes("vol-csi", "vol-boot", "vol-manual", "vol-extra"),
		}, nil)
		if _, err := client.StorageV1().CSIDrivers().Create(t.Context(), testCSIDriver(new(periodSeconds)), metav1.CreateOptions{}); err != nil {
			t.Fatalf("create CSIDriver: %v", err)
		}
		// The only installed node gets the first slot, so it is due right away.
		synctest.Wait()
		if got := provider.callCount(); got <= calls {
			t.Errorf("ControllerGetNodeInfo calls after re-enabling periodic updates: got %d, want more than %d", got, calls)
		}
		if got := ptr.Deref(getDriver(t).Allocatable.Count, -1); got != 22 {
			t.Errorf("allocatable count after re-enabling periodic updates: got %d, want 22", got)
		}
		if events := drainEvents(recorder); len(events) != 1 || !strings.Contains(events[0], "CSINodeInfoUpdated") {
			t.Errorf("events: got %v, want one CSINodeInfoUpdated", events)
		}
	})
}
