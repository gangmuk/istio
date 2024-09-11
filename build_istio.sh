#!/bin/bash

build_output_dir="build-output/gangmuk-envoy"
echo "project_repo: ${project_repo}"
echo "build_output_dir: ${build_output_dir}"

if [ -z "${project_repo}" ]; then
  echo "project_repo is empty"
  echo "exit..."
  exit
fi
if [ -z "${build_output_dir}" ]; then
  echo "build_output_dir is empty"
  echo "exit..."
  exit
fi
if [ -z "${docker_id}" ]; then
  echo "docker_id is empty"
  echo "exit..."
  exit
fi
if [ -z "${docker_pw}" ]; then
  echo "docker_pw is empty"
  echo "exit..."
  exit
fi
echo "Will start in 5 seconds"
sleep 5

echo ${docker_pw} | docker login --username ${docker_id} --password-stdin
echo "****** Docker login"

cd ${project_repo}/istio && \
sudo make docker && \
sudo make build &&
echo "****** Istio make docker and make build"

## Copy custom envoy binary to istio out directory
sudo cp ${project_repo}/proxy/${build_output_dir}/envoy ${project_repo}/istio/out/linux_amd64/release/
# sudo cp ${project_repo}/proxy/${build_output_dir}/bin/envoy ${project_repo}/istio/out/linux_amd64/release/
echo "****** Copy Envoy binary to Istio out directory"

## Push
cd ${project_repo}/istio && \
sudo make docker.proxyv2 && \
sudo make push.docker.proxyv2
echo "****** Istio with the new envoy is built and pushed"

## Install
cd ${project_repo}/istio && \
go run ${PWD}/istioctl/cmd/istioctl install --set hub=docker.io/${docker_id}  --set tag=latest -y
echo "****** Istio is installed"

echo "****** Restart everything!"
krrdistio
krrd