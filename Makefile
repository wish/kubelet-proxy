SHELL := /bin/bash

REPO_ROOT := $(CURDIR)

CURL = $(shell which curl 2> /dev/null)
WGET = $(shell which wget 2> /dev/null)

UNAME = $(shell uname | tr A-Z a-z)
ifeq ($(UNAME),)
$(error "Unable to determine OS type")
endif

ifeq ($(CURL)$(WGET),)
$(error "Neither curl nor wget available")
endif

include make/*.mk

.PHONY: integration_tests
integration_tests: build_dev_image cluster_start copy_dev_image run_integration_tests

# the INTEGRATION_TEST_FLAGS env var can be set to eg run only specific tests, e.g.:
# INTEGRATION_TEST_FLAGS='-test.run TestHappyPathWithMetrics' make run_integration_tests
.PHONY: run_integration_tests
run_integration_tests:
	@ echo "### Starting integration tests with Kubernetes version: $(KUBERNETES_VERSION) ###"
	cd integration_tests && KUBECONFIG=$(KUBECONFIG) KUBECTL=$(KUBECTL) go test -count 1 -v $(INTEGRATION_TEST_FLAGS)
