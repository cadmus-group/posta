MODULE := github.com/cadmus-group/posta
BIN_DIR ?= bin

VERSION ?= $(shell git describe --tags --match 'v*' --abbrev=0 2>/dev/null || echo v0.0.0-dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LD_FLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

CGO_ENABLED ?= 0
GO_BUILD := CGO_ENABLED=$(CGO_ENABLED) go build -trimpath -ldflags "$(LD_FLAGS)"

IMAGE_REPO ?= ghcr.io/cadmus-group/posta
IMAGE_TAG ?= $(VERSION)
IMAGE_QUEUE_API ?= $(IMAGE_REPO)/queue-api:$(IMAGE_TAG)
IMAGE_WORKER ?= $(IMAGE_REPO)/worker:$(IMAGE_TAG)
IMAGE_SCHEDULER ?= $(IMAGE_REPO)/scheduler:$(IMAGE_TAG)

.PHONY: build build-queue-api build-worker build-scheduler test vet \
	docker-all docker-queue-api docker-worker docker-scheduler version \
	gen-openapi-go gen-clients

build: build-queue-api build-worker build-scheduler

build-queue-api:
	$(GO_BUILD) -o $(BIN_DIR)/queue-api ./cmd/queue-api

build-worker:
	$(GO_BUILD) -o $(BIN_DIR)/worker ./cmd/worker

build-scheduler:
	$(GO_BUILD) -o $(BIN_DIR)/scheduler ./cmd/scheduler

test:
	go test ./...

vet:
	go vet ./...

docker-all: docker-queue-api docker-worker docker-scheduler

docker-queue-api:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -f Dockerfile.queue-api -t $(IMAGE_QUEUE_API) .

docker-worker:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -f Dockerfile.worker -t $(IMAGE_WORKER) .

docker-scheduler:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_DATE=$(BUILD_DATE) -f Dockerfile.scheduler -t $(IMAGE_SCHEDULER) .

version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "BuildDate: $(BUILD_DATE)"

gen-openapi-go:
	./scripts/gen-openapi-go.sh

gen-clients:
	./scripts/gen-clients.sh

