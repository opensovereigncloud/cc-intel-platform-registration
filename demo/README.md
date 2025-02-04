# Visualizing registration metrics with Prometheus and Grafana

This demonstration shows the `cc-intel-platform-registration` in action.

We are going to deploy a simple Grafana + Prometheus setup to help with the visualization of the metrics exported by `cc-intel-platform-registration`.

## Step 1: deploy `cc-intel-platform-registration`

Assuming that you have access to a k8s cluster via `kubectl`, and docker and `helm` installed,

- (1) build the image of `cc-intel-platform-registration`
- (2) push the image into a registry (or use `k3d image import`, if using k3d dev cluster)
- (3) create a pull secret for the HELM release
- (4) install `cc-intel-platform-registration` using `helm`

To do so, execute the following commands, assuming you have cloned and are in the root directory of this repository:

```bash
# (1)
export REGISTRY=your-registry.com/some-path # replace this by the registry path in which you have write permissions
export VERSION=demo
make build-image DOCKER_REGISTRY=$REGISTRY VERSION=$VERSION

# (2)
docker push $REGISTRY/cc-intel-platform-registration:$VERSION

# (3)
kubectl create namespace reg-svc-demo
export REGISTRY_USERNAME=<your-name> # username of the registry service account
export REGISTRY_ACCESS_TOKEN=<your-access-toke> # read access token
export REGISTRY_EMAIL=<your-email> # email address of the service account
kubectl create secret docker-registry reg-svc-pull-secret \
   --docker-server=registry.scontain.com \
   --docker-username=$REGISTRY_USERNAME \
   --docker-password=$REGISTRY_ACCESS_TOKEN \
   --docker-email=$REGISTRY_EMAIL \
   --namespace reg-svc-demo

# (4)
helm install reg-svc --namespace reg-svc-demo helm-chart/ \
    --set "fullnameOverride=reg-svc" \
    --set imagePullSecrets[0].name=reg-svc-pull-secret
```

After final step, ensure the registration service pod is ready:

```bash
kubectl get pod -n reg-svc-demo
```

## Step 2: deploy Grafana dashboard

Execute the `run-demo.sh` script to install a simple Grafana + Prometheus setup. Look into the `demo-manifests` directory to check the k8s manifests.

All the configurables and dashboard are already set for the values used in this demonstration.

```bash
cd demo
bash run-demo.sh
```

If execution is successful, you should see the default credentials for Grafana:

```
Deployment completed!
Default Grafana credentials:
Username: admin
Password: admin
```

Check that all pods are ready.

```bash
kubectl get pod -n monitoring
```

## Step 3: check grafana dashboard

Use `kubectl` port forwarding to have local access to the Grafana UI.

```bash
kubectl port-forward -n monitoring services/grafana 3000
```

Then, use your browser to nativate to [localhost:3000](localhost:3000). User is `admin` and password is `admin`. You can skip the updade of password in this demonstration.

After login, click on dashboards on the left side menu and check the Registration Service dashboard out.
