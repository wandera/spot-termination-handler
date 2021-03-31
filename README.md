# Welcome to spot-termination-handler

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
