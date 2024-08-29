#!/bin/bash
sudo cp ../proxy/bazel-out/k8-opt/bin/envoy out/linux_amd64/release/
pat=$1

docker login ghcr.io -u gangmuk -p ${pat}
sudo make docker
sudo make build

sudo cp ../proxy/build-output/envoy-gangmuk/bin/envoy ./out/linux_amd64/release/

sudo make docker.proxyv2
sudo make push.docker.proxyv2 # This one requires docker login


## These two lines are not required again if you did it before
# sudo make docker.pilot
# sudo docker push docker.io/gangmuk/pilot:latest

go run ./istioctl/cmd/istioctl install --set hub=docker.io/gangmuk  --set tag=latest -y
