/*
Copyright 2025 The Kubernetes Authors.

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

package features

import (
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/component-base/featuregate"
)

const (
	// owner: @torredil @gnufied @msau42
	// kep: https://kep.k8s.io/4876
	// alpha: v1.33
	// beta: v1.34
	//
	// Makes CSINode.Spec.Drivers[*].Allocatable.Count mutable, allowing CSI drivers to
	// update the number of volumes that can be allocated on a node. Additionally, enables
	// setting ErrorCode field in VolumeAttachment status.
	MutableCSINodeAllocatableCount featuregate.Feature = "MutableCSINodeAllocatableCount"
)

func init() {
	feature.DefaultMutableFeatureGate.Add(defaultKubernetesFeatureGates)
}

// defaultKubernetesFeatureGates consists of all known feature keys specific to external-attacher.
// To add a new feature, define a key for it above and add it here.
var defaultKubernetesFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	MutableCSINodeAllocatableCount: {Default: false, PreRelease: featuregate.Beta},
}
