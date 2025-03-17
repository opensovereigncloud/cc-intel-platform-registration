#!/bin/bash

set -e

pushd demo-manifests
kubectl apply -f namespace.yaml
kubectl apply -f prometheus.yaml
kubectl apply -f grafana.yaml
popd

echo "Deployment completed!"
echo "Default Grafana credentials:"
echo "Username: admin"
echo "Password: admin"
