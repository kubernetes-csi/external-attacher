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
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
)

// This is an unit test framework. It is heavily inspired by serviceaccount
// controller tests.

type reaction struct {
	verb     string
	resource string
	reactor  func(t *testing.T) core.ReactionFunc
}

type testCase struct {
	// Name of the test (for logging)
	name string
	// Object to insert into fake kubeclient before the test starts.
	initialObjects []runtime.Object
	// Optional client reactors
	reactors []reaction
	// Optional VolumeAttachment that's used to simulate "VA added" event.
	// This VA is also automatically added to initialObjects.
	addedVa *storagev1.VolumeAttachment
	// Optional VolumeAttachment that's used to simulate "VA updated" event.
	// This VA is also automatically added to initialObjects.
	updatedVa *storagev1.VolumeAttachment
	// List of expected kubeclient actions that should happen during the test.
	expectedActions []core.Action
}

func runTests(t *testing.T, tests []testCase) {
	for _, test := range tests {
		glog.Infof("Test %q: started", test.name)
		objs := test.initialObjects
		if test.addedVa != nil {
			objs = append(objs, test.addedVa)
		}
		if test.updatedVa != nil {
			objs = append(objs, test.updatedVa)
		}
		client := fake.NewSimpleClientset(objs...)
		informers := informers.NewSharedInformerFactory(client, time.Hour /* disable resync*/)
		vaInformer := informers.Storage().V1().VolumeAttachments()
		handler := NewTrivialHandler(client)

		for _, reactor := range test.reactors {
			client.Fake.PrependReactor(reactor.verb, reactor.resource, reactor.reactor(t))
		}

		ctrl := NewCSIAttachController(client, testAttacherName, handler, vaInformer)
		if test.addedVa != nil {
			vaInformer.Informer().GetStore().Add(test.addedVa)
			ctrl.vaAdded(test.addedVa)
		}
		if test.updatedVa != nil {
			vaInformer.Informer().GetStore().Update(test.updatedVa)
			ctrl.vaUpdated(test.updatedVa, test.updatedVa)
		}

		/* process the queue until we get expected results */
		timeout := time.Now().Add(10 * time.Second)
		lastReportedActionCount := 0
		for {
			if time.Now().After(timeout) {
				t.Errorf("Test %q: timed out", test.name)
				break
			}
			if ctrl.queue.Len() > 0 {
				glog.V(4).Infof("Test %q: %d events in the queue, processing one", test.name, ctrl.queue.Len())
				ctrl.processNextWorkItem()
			}
			if ctrl.queue.Len() > 0 {
				// There is still some work in the queue, process it now
				continue
			}
			currentActionCount := len(client.Actions())
			if currentActionCount < len(test.expectedActions) {
				if lastReportedActionCount < currentActionCount {
					glog.V(4).Infof("Test %q: got %d actions out of %d, waiting for the rest", test.name, currentActionCount, len(test.expectedActions))
					lastReportedActionCount = currentActionCount
				}
				// The test expected more to happen, wait for them
				time.Sleep(10 * time.Millisecond)
				continue
			}
			break
		}

		actions := client.Actions()
		for i, action := range actions {
			if len(test.expectedActions) < i+1 {
				t.Errorf("Test %q: %d unexpected actions: %+v", test.name, len(actions)-len(test.expectedActions), actions[i:])
				break
			}

			expectedAction := test.expectedActions[i]
			if !reflect.DeepEqual(expectedAction, action) {
				t.Errorf("Test %q:\nExpected:\n%s\ngot:\n%s", test.name, spew.Sdump(expectedAction), spew.Sdump(action))
				continue
			}
		}

		if len(test.expectedActions) > len(actions) {
			t.Errorf("Test %q: %d additional expected actions", test.name, len(test.expectedActions)-len(actions))
			for _, a := range test.expectedActions[len(actions):] {
				t.Logf("    %+v", a)
			}
		}
		glog.Infof("Test %q: finished \n\n", test.name)
	}
}
