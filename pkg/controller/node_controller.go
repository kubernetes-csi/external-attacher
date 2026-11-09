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
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	storagev1 "k8s.io/api/storage/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	storageinformers "k8s.io/client-go/informers/storage/v1"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

type NodeInfoHandler interface {
	// HasSynced reports whether the informer caches the handler reads are populated.
	// SyncNode must not run before that, otherwise it works on a partial view
	// and can compute a wrong node allocatable count.
	HasSynced() bool
	// SyncNode installs or refreshes the driver's CSINode.Spec.Drivers entry and node labels.
	// nodeID is the value from csiNode.Spec.DriverRegistrations for this driver;
	// the caller guarantees the registration exists.
	SyncNode(ctx context.Context, csiNode *storagev1.CSINode, nodeID string, refresh bool) error
}

type nodeState struct {
	scheduledAt time.Time      // time of next periodical update
	requests    []chan<- error // if non-nil, trigger an immediate update
}

// NodeController handles ControllerGetNodeInfo on driver registration and
// refresh periodically or on demand.
type NodeController struct {
	driverName string

	nodeQueue workqueue.TypedRateLimitingInterface[string]

	handler           NodeInfoHandler
	csiNodeLister     storagelisters.CSINodeLister
	csiNodeSynced     cache.InformerSynced
	csiDriverInformer storageinformers.CSIDriverIndexInformer

	stateMu      sync.Mutex    // protects updatePeriod and state
	updatePeriod time.Duration // zero if update is disabled
	state        map[string]nodeState
}

// NewNodeController returns a new *CSINodeController
func NewNodeController(
	logger klog.Logger,
	driverName string,
	handler NodeInfoHandler,
	csiNodeInformer storageinformers.TypedCSINodeInformer,
	csiDriverInformer storageinformers.TypedCSIDriverInformer,
	nodeRateLimiter workqueue.TypedRateLimiter[string],
) *NodeController {
	ctrl := &NodeController{
		driverName:        driverName,
		handler:           handler,
		csiNodeLister:     csiNodeInformer.Lister(),
		csiNodeSynced:     csiNodeInformer.TypedInformer().HasSynced,
		csiDriverInformer: csiDriverInformer.TypedInformer(),
		state:             map[string]nodeState{},
		nodeQueue: workqueue.NewTypedRateLimitingQueueWithConfig(nodeRateLimiter,
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "csi-attacher-node"}),
	}

	_, err := csiNodeInformer.TypedInformer().AddTypedEventHandler(cache.TypedResourceEventHandlerFuncs[*storagev1.CSINode]{
		AddFunc:    func(obj *storagev1.CSINode) { ctrl.nodeQueue.Add(obj.Name) },
		UpdateFunc: func(old, new *storagev1.CSINode) { ctrl.nodeQueue.Add(new.Name) },
		DeleteFunc: func(obj cache.DeletedObject[*storagev1.CSINode]) { ctrl.nodeQueue.Add(obj.GetName()) },
	}, cache.HandlerOptions{Logger: &logger})
	if err != nil {
		panic(err)
	}
	return ctrl
}

// Run starts CSI attacher and listens on channel events
func (ctrl *NodeController) Run(ctx context.Context, workers int) {
	var wg sync.WaitGroup
	defer wg.Wait()
	defer ctrl.nodeQueue.ShutDown()

	logger := klog.FromContext(ctx)
	logger.Info("Starting CSI attacher node controller")
	defer logger.Info("Shutting CSI attacher node controller")

	if !cache.WaitForCacheSync(ctx.Done(), ctrl.csiNodeSynced) {
		logger.Error(nil, "Cannot sync caches")
		return
	}
	// Add CSIDriver handler after CSINode store synced, it needs full list of nodes.
	csiDriverHandler, err := ctrl.csiDriverInformer.AddTypedEventHandler(cache.TypedResourceEventHandlerFuncs[*storagev1.CSIDriver]{
		AddFunc: func(obj *storagev1.CSIDriver) {
			if obj.Name == ctrl.driverName {
				ctrl.syncUpdatePeriod(logger, obj.Spec.NodeAllocatableUpdatePeriodSeconds)
			}
		},
		UpdateFunc: func(old, new *storagev1.CSIDriver) {
			if new.Name == ctrl.driverName {
				ctrl.syncUpdatePeriod(logger, new.Spec.NodeAllocatableUpdatePeriodSeconds)
			}
		},
		DeleteFunc: func(obj cache.DeletedObject[*storagev1.CSIDriver]) {
			if obj.GetName() == ctrl.driverName {
				ctrl.syncUpdatePeriod(logger, nil)
			}
		},
	}, cache.HandlerOptions{Logger: &logger})
	if err != nil {
		logger.Error(err, "Cannot add CSIDriver event handler")
		return
	}

	// Wait for the initial update scheduled, and the caches read by the handler.
	if !cache.WaitForCacheSync(ctx.Done(), csiDriverHandler.HasSynced, ctrl.handler.HasSynced) {
		logger.Error(nil, "Cannot sync caches")
		return
	}

	for range workers {
		wg.Go(func() {
			wait.UntilWithContext(ctx, ctrl.syncNode, 0)
		})
	}
	<-ctx.Done()
}

func (ctrl *NodeController) syncNode(ctx context.Context) {
	nodeName, quit := ctrl.nodeQueue.Get()
	if quit {
		return
	}
	defer ctrl.nodeQueue.Done(nodeName)

	logger := klog.LoggerWithValues(klog.FromContext(ctx), "Node", nodeName)
	ctx = klog.NewContext(ctx, logger)
	logger.V(4).Info("Started Node processing")

	ctrl.stateMu.Lock()
	state := ctrl.state[nodeName]
	requests := state.requests
	state.requests = nil
	ctrl.state[nodeName] = state
	ctrl.stateMu.Unlock()

	now := time.Now()
	refresh := now.After(state.scheduledAt) || requests != nil

	managed, err := ctrl.processNode(ctx, nodeName, refresh)
	for _, r := range requests {
		r <- err
		close(r)
	}

	if err != nil {
		logger.V(1).Error(err, "Error sync Node")
		ctrl.nodeQueue.AddRateLimited(nodeName)
		return
	}
	ctrl.nodeQueue.Forget(nodeName)

	ctrl.stateMu.Lock()
	state = ctrl.state[nodeName]
	if state.requests != nil || state.scheduledAt.After(now) {
		// New request since we start process, workqueue is dirty.
		// Or not refreshing or [syncUpdatePeriod] already scheduled next refresh.
	} else if !managed || ctrl.updatePeriod <= 0 {
		delete(ctrl.state, nodeName)
	} else {
		if state.scheduledAt.IsZero() {
			// For the first update, use a random period in [0, updatePeriod) to distribute them uniformly.
			// Expectation is updatePeriod/2, shorter but consisitent with [syncUpdatePeriod].
			// bring cluster load to steady state immediately
			state.scheduledAt = now.Add(rand.N(ctrl.updatePeriod))
		} else {
			// Add a small jitter to keep update spreaded.
			state.scheduledAt = now.Add(ctrl.updatePeriod + rand.N(ctrl.updatePeriod/25))
		}
		ctrl.state[nodeName] = state
		duration := time.Until(state.scheduledAt)
		ctrl.nodeQueue.AddAfter(nodeName, duration)
		logger.V(3).Info("Node info update scheduled", "in", duration)
	}
	ctrl.stateMu.Unlock()
}

func (ctrl *NodeController) processNode(ctx context.Context, nodeName string, refresh bool) (bool, error) {
	logger := klog.FromContext(ctx)
	csiNode, err := ctrl.csiNodeLister.Get(nodeName)
	if err != nil {
		if apierrs.IsNotFound(err) {
			logger.V(3).Info("Node was deleted, ignoring")
			return false, nil
		}
		return false, fmt.Errorf("get node from informer: %w", err)
	}

	var nodeID string
	for _, r := range csiNode.Spec.DriverRegistrations {
		if r.Name == ctrl.driverName {
			nodeID = r.NodeID
		}
	}
	if nodeID == "" {
		logger.V(4).Info("Not registered")
		return false, nil
	}
	ctx = klog.NewContext(ctx, klog.LoggerWithValues(logger, "nodeID", nodeID))
	return true, ctrl.handler.SyncNode(ctx, csiNode, nodeID, refresh)
}

func (ctrl *NodeController) isRegistered(csiNode *storagev1.CSINode) bool {
	return slices.ContainsFunc(csiNode.Spec.DriverRegistrations, func(r storagev1.CSINodeDriverRegistration) bool {
		return r.Name == ctrl.driverName
	})
}

func (ctrl *NodeController) isInstalled(csiNode *storagev1.CSINode) bool {
	return slices.ContainsFunc(csiNode.Spec.Drivers, func(d storagev1.CSINodeDriver) bool {
		return d.Name == ctrl.driverName
	})
}

func (ctrl *NodeController) syncUpdatePeriod(logger klog.Logger, periodSeconds *int64) {
	period := time.Duration(ptr.Deref(periodSeconds, 0)) * time.Second

	var nodes []*storagev1.CSINode
	if periodSeconds != nil {
		var err error
		nodes, err = ctrl.csiNodeLister.List(labels.Everything())
		if err != nil {
			logger.Error(err, "List CSINodes from informer")
			return
		}
	}

	ctrl.stateMu.Lock()
	defer ctrl.stateMu.Unlock()

	ctrl.updatePeriod = period
	if len(nodes) > 0 {
		// (Re)schedule new nodes or nodes that was scheduled too far away.
		// Evenly distribute them in the period. This is nice because:
		// - cluster load reaches steady state immediately after controller restart or config change.
		// - the expectation of the residual age of a uniformly-phased periodic update is updatePeriod/2,
		//   so this keeps the expectation between update at updatePeriod even across controller restart.
		now := time.Now()
		bound := now.Add(period)
		newNodes := make([]string, 0, len(nodes))
		for _, n := range nodes {
			if !ctrl.isRegistered(n) || !ctrl.isInstalled(n) {
				continue
			}
			scheduled := ctrl.state[n.Name]
			if scheduled.scheduledAt.IsZero() || scheduled.scheduledAt.After(bound) {
				newNodes = append(newNodes, n.Name)
			}
		}
		if len(newNodes) == 0 {
			return
		}

		rand.Shuffle(len(newNodes), func(i, j int) {
			newNodes[i], newNodes[j] = newNodes[j], newNodes[i]
		})

		step := period / time.Duration(len(newNodes))
		for i, nodeName := range newNodes {
			after := time.Duration(i) * step
			s := ctrl.state[nodeName]
			s.scheduledAt = now.Add(after)
			ctrl.state[nodeName] = s
			ctrl.nodeQueue.AddAfter(nodeName, after)
		}
	}
}

func (ctrl *NodeController) RequestUpdate(ctx context.Context, nodeName string) error {
	ctrl.stateMu.Lock()
	if ctrl.updatePeriod <= 0 {
		ctrl.stateMu.Unlock()
		return nil
	}

	// Buffer the answer: the requester may give up on ctx.Done() and never receive it.
	req := make(chan error, 1)
	state := ctrl.state[nodeName]
	state.requests = append(state.requests, req)
	ctrl.state[nodeName] = state
	ctrl.stateMu.Unlock()

	ctrl.nodeQueue.Add(nodeName)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-req:
		return err
	}
}
