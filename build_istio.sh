#!/bin/bash

docker login ghcr.io -u ${github_id} -p ${PAT}

echo ${docker_pw} | docker login --username ${docker_id} --password-stdin && docker push gangmuk/proxyv2:latest

sudo make docker
sudo make build

# project_repo="/proj/istio-PG0/projects"
sudo cp ${project_repo}/proxy/build-output/envoy-gangmuk/bin/envoy ./out/linux_amd64/release/

sudo make docker.proxyv2
sudo make push.docker.proxyv2 # This one requires docker login

## These two lines are not required again if you did it before
# sudo make docker.pilot
# sudo docker push docker.io/gangmuk/pilot:latest

