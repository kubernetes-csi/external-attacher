# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: all csi-attacher clean test

GCR_IMAGE_PATH=gcr.io/google-containers/sig-storage/csi/external-attacher
DOCKER_IMAGE_PATH=docker.io/k8scsi/csi-attacher

IMAGE_VERSION :=
TAG := $(shell git describe --abbrev=0 --tags HEAD 2>/dev/null)
COMMIT := $(shell git rev-parse HEAD)
ifeq ($(TAG),)
    IMAGE_VERSION := latest
else
    ifeq ($(COMMIT), $(shell git rev-list -n1 $(TAG)))
        IMAGE_VERSION := $(TAG)
    else
        IMAGE_VERSION := $(TAG)-$(COMMIT)
    endif
endif

ifdef V
TESTARGS = -v -args -alsologtostderr -v 5
else
TESTARGS = 
endif


all: csi-attacher

csi-attacher:
	go build -o csi-attacher cmd/csi-attacher/main.go

clean:
	-rm -rf csi-attacher deploy/docker/csi-attacher

container: csi-attacher
	cp csi-attacher deploy/docker
	docker build -t $(DOCKER_IMAGE_PATH):$(IMAGE_VERSION) deploy/docker

push: container
	docker push $(DOCKER_IMAGE_PATH):$(IMAGE_VERSION)

gcr-push: csi-attacher
	cp bin/csi-attacher deploy/docker
	docker build -t $(GCR_IMAGE_PATH):$(IMAGE_VERSION) deploy/docker
	gcloud docker -- push $(GCR_IMAGE_PATH):$(IMAGE_VERSION)

test:
	go test `go list ./... | grep -v 'vendor'` $(TESTARGS)
	go vet `go list ./... | grep -v vendor`
