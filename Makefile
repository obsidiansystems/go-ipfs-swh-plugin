# Force Go Modules
GO111MODULE = on

GOCC ?= go
GOFLAGS ?=

# If set, override the install location for plugins
IPFS_PATH ?= $(HOME)/.ipfs

# If set, override the IPFS version to build against. This _modifies_ the local
# go.mod/go.sum files and permanently sets this version.
IPFS_VERSION ?= $(lastword $(shell $(GOCC) list -m github.com/ipfs/kubo))

# make reproducible
ifneq ($(findstring /,$(IPFS_VERSION)),)
# Locally built kubo
GOFLAGS += -asmflags=all=-trimpath="$(GOPATH)" -gcflags=all=-trimpath="$(GOPATH)"
else
# Remote version of kubo (e.g. via `go get -trimpath` or official distribution)
GOFLAGS += -trimpath
endif

.PHONY: install build

go.mod:
	bash ./build/set-target.sh $(IPFS_VERSION)

swhid-plugin.so: main/main.go go.mod
	$(GOCC) build $(GOFLAGS) -buildmode=plugin -o "$@" "$<"
	chmod +x "$@"

build: swhid-plugin.so
	@echo "Built against" $(IPFS_VERSION)

install: build
	mkdir -p "$(IPFS_PATH)/plugins/"
	cp -f swhid-plugin.so "$(IPFS_PATH)/plugins/swhid-plugin.so"
