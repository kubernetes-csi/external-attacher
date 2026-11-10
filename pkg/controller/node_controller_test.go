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
	"strings"
	"testing"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2/ktesting"
)

type fakeNodeInfoHandler struct {
	err       error
	syncCalls int
}

func (h *fakeNodeInfoHandler) HasSynced() bool { return true }

func (h *fakeNodeInfoHandler) SyncNode(ctx context.Context, csiNode *storagev1.CSINode, nodeID string, refresh bool) error {
	h.syncCalls++
	return h.err
}

func newTestNodeController(t *testing.T, handler NodeInfoHandler, csiNode *storagev1.CSINode) (*NodeController, *record.FakeRecorder) {
	t.Helper()
	factory := informers.NewSharedInformerFactory(fake.NewClientset(), 0)
	csiNodeInformer := factory.Storage().V1().CSINodes()
	if err := csiNodeInformer.Informer().GetStore().Add(csiNode); err != nil {
		t.Fatalf("add CSINode to store: %v", err)
	}
	recorder := record.NewFakeRecorder(10)
	logger, _ := ktesting.NewTestContext(t)
	ctrl := NewNodeController(
		logger,
		testAttacherName,
		handler,
		recorder,
		csiNodeInformer,
		factory.Storage().V1().CSIDrivers(),
		workqueue.DefaultTypedControllerRateLimiter[string](),
	)
	return ctrl, recorder
}

func TestProcessNodeEvents(t *testing.T) {
	testcases := []struct {
		name       string
		handlerErr error
		wantEvents int
	}{
		{name: "success", handlerErr: nil, wantEvents: 0},
		{name: "failure", handlerErr: errors.New("get node info: boom"), wantEvents: 1},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			handler := &fakeNodeInfoHandler{err: tc.handlerErr}
			ctrl, recorder := newTestNodeController(t, handler, testCSINode("node-uid"))

			managed, err := ctrl.processNode(t.Context(), testNodeName, false)
			if tc.handlerErr != nil && err == nil {
				t.Fatal("expected error")
			}
			if tc.handlerErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !managed || handler.syncCalls != 1 {
				t.Errorf("managed: got %v, sync calls: got %d", managed, handler.syncCalls)
			}

			events := drainEvents(recorder)
			if len(events) != tc.wantEvents {
				t.Fatalf("events: got %v, want %d", events, tc.wantEvents)
			}
			if tc.handlerErr == nil {
				return
			}
			for _, want := range []string{"Warning", "CSINodeInfoUpdateFailed", testAttacherName, tc.handlerErr.Error()} {
				if !strings.Contains(events[0], want) {
					t.Errorf("event %q does not contain %q", events[0], want)
				}
			}
		})
	}
}

func TestProcessNodeNotRegistered(t *testing.T) {
	handler := &fakeNodeInfoHandler{}
	csiNode := testCSINode("node-uid")
	csiNode.Spec.DriverRegistrations = nil
	ctrl, recorder := newTestNodeController(t, handler, csiNode)

	managed, err := ctrl.processNode(t.Context(), testNodeName, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if managed || handler.syncCalls != 0 {
		t.Errorf("managed: got %v, sync calls: got %d", managed, handler.syncCalls)
	}
	if events := drainEvents(recorder); len(events) != 0 {
		t.Errorf("unexpected events: %v", events)
	}
}
