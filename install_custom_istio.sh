#!/bin/bash

# project_repo="/proj/istio-PG0/projects"
go run ${PWD}/istioctl/cmd/istioctl install --set hub=docker.io/gangmuk  --set tag=latest -y

# Restart everything
krrdistio
krrd
