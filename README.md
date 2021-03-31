# Welcome to spot-termination-handler
[![Docker](https://github.com/wandera/spot-termination-handler/actions/workflows/docker.yml/badge.svg?branch=master)](https://github.com/wandera/spot-termination-handler/actions/workflows/docker.yml)
[![Docker Pulls](https://img.shields.io/docker/pulls/wanderadock/spot-termination-handler)](https://hub.docker.com/repository/docker/wanderadock/spot-termination-handler)

## Description
Spot-termination-handler app monitors EC2 spot instance for termination event.

## Docker image
```bash
wanderadock/spot-termination-handler:latest
```

## Deployment
The suggested deployment method is a DaemonSet running only on spot nodes.
Check [examples](./examples) folder.

## Configuration
* `POD_NAME` - Name of the Pod running the termination handler (itself), source for K8s events (required)
* `NODE_NAME` - Node name on which the termination handler is running (required)
* `FORCE` - If to force Pod deletion (default: true)
* `DELETE_EMPTY_DIR` - If to delete empty dirs (default: true)
* `IGNORE_DAEMONSETS` - If to ignore daemonset pods during drain (default: true)
* `GRACE_PERIOD` - Grace period (in seconds) given to each pod (default: 120)
* `DEV_MODE` - Dev mode for local development (default: false)
* `LOG_LEVEL` - Log level (default: DEBUG)

## Development environment prerequisites
* [Go](https://golang.org/) >= 1.16
* Docker (optional)

## Prepare environment properties
* configure NODE_NAME - name of the node that will be drained
  * `NODE_NAME=kind-control-plane`
* configure POD_NAME - id of the spot-termination-handler pod
  * `POD_NAME=spot-termination-handler`

## Starting the application
* build and start app `make run`
