# Welcome to ppot-termination-handler

Spot-termination-handler app monitors EC2 spot instance for termination event. 

## Development environment prerequisites
* [Go](https://golang.org/) >= 1.16
* Docker (optional)

## Prepare environment properties
* configure kubectl drain parameters - default `'--grace-period=120 --force --ignore-daemonsets'`
    * `DRAIN_PARAMETERS=`

## Starting the application
* build and start app `make run`
