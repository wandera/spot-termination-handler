# Image URL to use all building/pushing image targets
IMG ?= spot-termination-handler:latest

all: clean check test build
run: spot-termination-handler
	./spot-termination-handler

prepare:
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.38.0
endif

check: prepare
	@echo "Running check"
	golangci-lint run

test:
	CGO_ENABLED=0 go test ./...

build: spot-termination-handler

spot-termination-handler: *.go **/*.go go.mod go.sum
	CGO_ENABLED=0 go build -o spot-termination-handler

# Build the docker image
docker-build:
	docker build . -t ${IMG}

clean:
	rm -f spot-termination-handler
