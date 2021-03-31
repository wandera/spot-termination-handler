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
