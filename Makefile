VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT  := $(shell git log -1 --format='%H')

all: install

LD_FLAGS = -X github.com/strangelove-ventures/horcrux-proxy/cmd.Version=$(VERSION) \
	-X github.com/strangelove-ventures/horcrux-proxy/cmd.Commit=$(COMMIT)
LD_FLAGS += $(LDFLAGS)
LD_FLAGS := $(strip $(LD_FLAGS))

BUILD_FLAGS := -tags netgo -trimpath -ldflags '-s -w $(LD_FLAGS)'

build:
	@go build -mod readonly $(BUILD_FLAGS) -o build/ ./...

install:
	@go install -mod readonly $(BUILD_FLAGS) ./...

test:
	@go test -race -timeout 30m -mod readonly -v ./...

clean:
	rm -rf build

build-docker:
	docker build -t strangelove-ventures/horcrux:$(VERSION) -f ./docker/horcrux/Dockerfile .

.PHONY: all test clean build