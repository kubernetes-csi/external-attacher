/*
Copyright 2019 The Kubernetes Authors.

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

package attacher

import (
	"context"
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"google.golang.org/grpc"
)

type CSIVolumeLister struct {
	client csi.ControllerClient
}

// NewVolumeLister provides a new VolumeLister object.
func NewVolumeLister(conn *grpc.ClientConn) *CSIVolumeLister {
	return &CSIVolumeLister{
		client: csi.NewControllerClient(conn),
	}
}

func (a *CSIVolumeLister) ListVolumes(ctx context.Context) (map[string]([]string), error) {
	p := map[string][]string{}

	tok := ""
	for {
		rsp, err := a.client.ListVolumes(ctx, &csi.ListVolumesRequest{
			StartingToken: tok,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list volumes: %v", err)
		}

		for _, e := range rsp.Entries {
			p[e.GetVolume().VolumeId] = e.Status.PublishedNodeIds
		}
		tok = rsp.NextToken

		if len(tok) == 0 {
			break
		}
	}

	return p, nil
}
