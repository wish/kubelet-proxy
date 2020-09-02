# K8S version can be overridden
# see available versions at https://hub.docker.com/r/kindest/node/tags
KUBERNETES_VERSION ?= 1.19.0
# see https://github.com/kubernetes-sigs/kind/releases
KIND_VERSION = 0.8.1

DEV_DIR = $(REPO_ROOT)/dev
BIN_DIR = $(DEV_DIR)/bin

CLUSTER_NAME ?= kubelet-proxy-dev

KIND = $(BIN_DIR)/kind-$(KIND_VERSION)
KIND_URL = https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-$(UNAME)-amd64

KUBECTL = $(shell which kubectl 2> /dev/null)
KUBECTL_URL = https://storage.googleapis.com/kubernetes-release/release/v$(KUBERNETES_VERSION)/bin/$(UNAME)/amd64/kubectl

ifeq ($(KUBECTL),)
KUBECTL = $(BIN_DIR)/kubectl-$(KUBERNETES_VERSION)
endif

KUBECONFIG = $(DEV_DIR)/kubeconfig

# starts a new kind cluster (see https://github.com/kubernetes-sigs/kind)
.PHONY: cluster_start
cluster_start: $(KIND) $(KUBECTL)
	if ! $(KIND) get clusters | grep -E '\b$(CLUSTER_NAME)\b' > /dev/null; then \
		$(KIND) create cluster --name '$(CLUSTER_NAME)' --image 'kindest/node:v$(KUBERNETES_VERSION)' \
			&& $(KIND) get kubeconfig --name '$(CLUSTER_NAME)' > '$(KUBECONFIG)' \
			&& sleep 30; \
	fi

# removes the kind cluster
.PHONY: cluster_clean
cluster_clean: $(KIND)
	$(KIND) delete cluster --name '$(CLUSTER_NAME)'

$(KIND):
	mkdir -vp '$(BIN_DIR)'
ifeq ($(WGET),)
	$(CURL) -L $(KIND_URL) > $(KIND)
else
	$(WGET) -O $(KIND) $(KIND_URL)
endif
	chmod +x $(KIND)

$(KUBECTL):
	mkdir -vp '$(BIN_DIR)'
ifeq ($(WGET),)
	$(CURL) -L $(KUBECTL_URL) > $(KUBECTL)
else
	$(WGET) -O $(KUBECTL) $(KUBECTL_URL)
endif
	chmod +x $(KUBECTL)

DEV_IMAGE_NAME = kubelet-proxy-dev

.PHONY: build_dev_image
build_dev_image:
	docker build --platform linux/amd64 $(REPO_ROOT) -t $(DEV_IMAGE_NAME)

.PHONY: copy_dev_image
copy_dev_image: $(KIND)
	$(KIND) load docker-image $(DEV_IMAGE_NAME) --name '$(CLUSTER_NAME)'
