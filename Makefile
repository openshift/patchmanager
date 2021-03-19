all: build
.PHONY: all

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/images.mk \
	targets/openshift/deps.mk \
)

build-image:
	podman build --squash -f Dockerfile -t quay.io/openshift/patchmanager:v0.1
.PHONY: build-image

push-image:
	podman push quay.io/openshift/patchmanager:v0.1
