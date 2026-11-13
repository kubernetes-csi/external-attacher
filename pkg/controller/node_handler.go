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
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/external-attacher/v4/pkg/attacher"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	coreapply "k8s.io/client-go/applyconfigurations/core/v1"
	storageapply "k8s.io/client-go/applyconfigurations/storage/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	storageinformers "k8s.io/client-go/informers/storage/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

type lockedSet struct {
	mu  sync.Mutex
	set sets.Set[string]
}

type nodeCSIHandler struct {
	driverName string
	client     kubernetes.Interface
	provider   attacher.NodeInfoProvider
	recorder   record.EventRecorder
	timeout    time.Duration // zero if unbounded

	resolver  volumeSourceResolver
	vaIndexer cache.TypedIndexer[*storagev1.VolumeAttachment]
	synced    []cache.InformerSynced

	processingVolumes sync.Map // map[string]*lockedSet
}

type NodeCSIHandler interface {
	NodeInfoHandler
	VolumeProcessingRecorder
}

func NewNodeCSIHandler(
	driverName string,
	client kubernetes.Interface,
	provider attacher.NodeInfoProvider,
	timeout time.Duration,
	recorder record.EventRecorder,
	vaInformer storageinformers.VolumeAttachmentIndexInformer,
	pvInformer coreinformers.PersistentVolumeInformer,
	translator AttacherCSITranslator,
) NodeCSIHandler {
	err := vaInformer.AddTypedIndexers(cache.TypedIndexers[*storagev1.VolumeAttachment]{
		VA_NODE_NAME_INDEX: func(va *storagev1.VolumeAttachment) ([]string, error) {
			return []string{va.Spec.Attacher + "/" + va.Spec.NodeName}, nil
		},
	})
	if err != nil {
		panic(err)
	}
	return &nodeCSIHandler{
		driverName: driverName,
		client:     client,
		provider:   provider,
		recorder:   recorder,
		timeout:    timeout,
		vaIndexer:  vaInformer.GetTypedIndexer(),
		synced:     []cache.InformerSynced{vaInformer.HasSynced, pvInformer.Informer().HasSynced},
		resolver: volumeSourceResolver{
			pvLister:   pvInformer.Lister(),
			translator: translator,
		},
	}
}

const VA_NODE_NAME_INDEX = "va-node-name"

func (h *nodeCSIHandler) vaNodeNameIndexKey(nodeName string) string {
	return h.driverName + "/" + nodeName
}

func (h *nodeCSIHandler) HasSynced() bool {
	for _, synced := range h.synced {
		if !synced() {
			return false
		}
	}
	return true
}

func (h *nodeCSIHandler) SyncNode(ctx context.Context, csiNode *storagev1.CSINode, nodeID string, refresh bool) error {
	logger := klog.FromContext(ctx)
	nodeName := csiNode.Name

	idriver := slices.IndexFunc(csiNode.Spec.Drivers, func(r storagev1.CSINodeDriver) bool {
		return r.Name == h.driverName && r.NodeID == nodeID
	})
	if idriver >= 0 {
		if !refresh {
			logger.V(4).Info("Already installed")
			return nil
		}
	}

	driver := storageapply.CSINodeDriver().
		WithName(h.driverName).
		WithNodeID(nodeID)

	nodeInfo, err := h.queryNodeInfo(ctx, logger, nodeName, driver)
	if err != nil {
		return err
	}

	fieldManager := "external-attacher/" + h.driverName

	if idriver >= 0 {
		// Ignore topology update when refreshing, matching kubelet
		driver.WithTopologyKeys(csiNode.Spec.Drivers[idriver].TopologyKeys...)
	} else {
		node, err := h.client.CoreV1().Nodes().Get(ctx, nodeName, v1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get Node: %w", err)
		}

		if !slices.ContainsFunc(csiNode.OwnerReferences, func(r v1.OwnerReference) bool {
			return r.Kind == "Node" && r.UID == node.UID
		}) {
			return fmt.Errorf("CSINode is not owned by this Node (uid: %s)", node.UID)
		}

		segments := nodeInfo.AccessibleTopology.GetSegments()
		nodeDirty := false
		for k, v := range segments {
			if curVal, exists := node.Labels[k]; exists {
				if curVal != v {
					return fmt.Errorf("detected topology value collision: driver reported %q:%q but existing label is %q:%q", k, v, k, curVal)
				}
			} else {
				nodeDirty = true
			}
		}
		driver.TopologyKeys = slices.Sorted(maps.Keys(segments))
		if nodeDirty {
			// Unlike kubelet, here we use SSA, which can remove a label if it is solely owned by us.
			// Not forced apply, we don't overwrite different values.
			_, err = h.client.CoreV1().Nodes().Apply(ctx,
				coreapply.Node(nodeName).WithUID(node.UID).WithLabels(segments),
				v1.ApplyOptions{FieldManager: fieldManager})
			if err != nil {
				return fmt.Errorf("apply Node labels: %w", err)
			}
		}
	}

	spec := storageapply.CSINodeSpec().WithDrivers(driver)
	if idriver >= 0 {
		// Renewing an existing entry: skip the write if we already own exactly this spec.
		owned, err := storageapply.ExtractCSINode(csiNode, fieldManager)
		if err != nil {
			return fmt.Errorf("extract applied configuration: %w", err)
		}
		if apiequality.Semantic.DeepEqual(owned.Spec, spec) {
			logger.V(4).Info("CSINode driver entry is up to date")
			return nil
		}
	}

	// Only the fields in spec are managed and taken with Force, everything else in the entry
	// keeps belonging to its manager. In particular, SSA leaves Allocatable alone if another
	// manager took it over, so we cannot delete it when the driver stops reporting a limit.
	_, err = h.client.StorageV1().CSINodes().Apply(ctx,
		storageapply.CSINode(nodeName).WithUID(csiNode.UID).WithResourceVersion(csiNode.ResourceVersion).
			WithSpec(spec),
		v1.ApplyOptions{Force: true, FieldManager: fieldManager})
	if err != nil {
		if apierrs.IsConflict(err) {
			logger.V(2).Info("Conflicting update CSINode, wait for another event")
			return nil
		}
		return fmt.Errorf("apply CSINode: %w", err)
	}
	logger.V(3).Info("CSI driver applied", "driver", driver)
	reason := "CSINodeInfoUpdated"
	verb := "Updated"
	if idriver < 0 {
		reason = "CSINodeRegistered"
		verb = "Registered"
	}
	h.recorder.Eventf(csiNode, corev1.EventTypeNormal, reason,
		"%s CSINode driver %q: %s", verb, h.driverName, describeDriver(driver))
	return nil
}

func describeDriver(driver *storageapply.CSINodeDriverApplyConfiguration) string {
	count := "unlimited"
	if driver.Allocatable != nil && driver.Allocatable.Count != nil {
		count = fmt.Sprintf("%d", *driver.Allocatable.Count)
	}
	return fmt.Sprintf("nodeID=%q, topologyKeys=%v, allocatableCount=%s", ptr.Deref(driver.NodeID, ""), driver.TopologyKeys, count)
}

func (h *nodeCSIHandler) queryNodeInfo(ctx context.Context, logger klog.Logger, nodeName string, driver *storageapply.CSINodeDriverApplyConfiguration) (
	*csi.ControllerGetNodeInfoResponse, error) {

	nodeID := *driver.NodeID
	processing := &lockedSet{set: sets.New[string]()}
	h.processingVolumes.Store(nodeID, processing)
	defer h.processingVolumes.Delete(nodeID)

	nodeInfo, err := h.getNodeInfo(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("call ControllerGetNodeInfo: %w", err)
	}

	if nodeInfo.MaxVolumesPerNode > 0 {
		maxVolumes := nodeInfo.MaxVolumesPerNode
		if len(nodeInfo.PublishedVolumes) > 0 {
			csiVolumes, err := h.getManagedVolumes(logger, nodeName, nodeID)
			if err != nil {
				return nil, err
			}

			// Include the volumes that is just detached during the CSI RPC call
			processing.mu.Lock()
			csiVolumes = csiVolumes.Union(processing.set)
			processing.mu.Unlock()

			for _, vol := range nodeInfo.PublishedVolumes {
				if !csiVolumes.Has(vol.VolumeId) {
					logger.V(5).Info("Unmanaged volume", "volumeId", vol.VolumeId)
					maxVolumes--
				}
			}
		}
		// Publish the remaining capacity. 0 means "no volume can be attached", not unlimited:
		// the scheduler distinguishes an unset Count (unbounded) from a Count of 0.
		driver.Allocatable = storageapply.VolumeNodeResources().WithCount(int32(max(maxVolumes, 0)))
	}
	return nodeInfo, nil
}

// getNodeInfo calls ControllerGetNodeInfo, bounded by the configured node timeout.
func (h *nodeCSIHandler) getNodeInfo(ctx context.Context, nodeID string) (*csi.ControllerGetNodeInfoResponse, error) {
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}
	return h.provider.GetNodeInfo(ctx, nodeID)
}

// Gather all the external-attacher managed volumes.
func (h *nodeCSIHandler) getManagedVolumes(logger klog.Logger, nodeName, nodeID string) (sets.Set[string], error) {
	vas, err := h.vaIndexer.ByTypedIndex(VA_NODE_NAME_INDEX, h.vaNodeNameIndexKey(nodeName))
	if err != nil {
		return nil, fmt.Errorf("get VAs from indexer: %w", err)
	}
	csiVolumes := make(sets.Set[string], len(vas))
	for _, va := range vas {
		src, err := h.resolver.resolve(logger, va)
		if err != nil {
			return nil, fmt.Errorf("resolve source for VA %q: %w", va.Name, err)
		}
		handle, err := h.resolver.attachedVolumeID(src, nodeID)
		if err != nil {
			logger.Error(err, "Failed to repair volume handle", "volumeHandle", src.spec.CSI.VolumeHandle)
			handle = src.spec.CSI.VolumeHandle
		}
		csiVolumes.Insert(handle)
	}
	return csiVolumes, nil
}

func (h *nodeCSIHandler) RecordProcessing(nodeID, volumeID string) {
	processing, ok := h.processingVolumes.Load(nodeID)
	if ok {
		processing := processing.(*lockedSet)
		processing.mu.Lock()
		defer processing.mu.Unlock()
		processing.set.Insert(volumeID)
	}
}

type NodeFailingHandler struct{ Err error }

func (h NodeFailingHandler) HasSynced() bool { return true }

func (h NodeFailingHandler) SyncNode(ctx context.Context, csiNode *storagev1.CSINode, nodeID string, refresh bool) error {
	return h.Err
}
