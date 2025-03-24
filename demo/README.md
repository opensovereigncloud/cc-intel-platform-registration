# Visualizing registration metrics with Prometheus and Grafana

This demonstration shows the `cc-intel-platform-registration` in action.

We are going to deploy a simple Grafana + Prometheus setup to help with the visualization of the metrics exported by `cc-intel-platform-registration`.

### EFI System 

`cc-intel-platform-registration` needs read and write capabilities for SGX UEFI variables.  It must be able to mount the `/sys/firmware/efi/efivars` host path.

When creating your cluster with `k3d`, you should explicitly bind the `/sys/firmware/efi/efivars` volume to your nodes:
e.g A cluster with 1 server and 2 agent nodes

```bash
k3d cluster create my-cluster-name \
    --agents 2 \
    -v /sys/firmware/efi/efivars:/sys/firmware/efi/efivars@agent:0,1 \
    -v /sys/firmware/efi/efivars:/sys/firmware/efi/efivars@server:0 
```
### SGX Device Support 

The service requires  a `sgx.intel.com/enclave: 1` on Kubernetes. 
For deploying with docker-compose, it requires access to `/dev/sgx_enclave`

## Run the `cc-intel-platform-registration` demo

- (1) build the image of `cc-intel-platform-registration`
- (2) push the image into a registry (or use `k3d image import`, if using k3d dev cluster)
- (3) run the `./run-demo.sh` script and choose your deployment option. (for docker compose, ensure you can pull the built image)

To do so, execute the following commands, assuming you have cloned and are in the root directory of this repository:

```bash
# (1)
nano ./demo/.env.template # edit the environmental variables
cp ./demo/.env.template .env 
source .env
# (2)
make build-image IMAGE_REGISTRY=$REGISTRY VERSION=$VERSION
# (3)
docker push $REGISTRY/cc-intel-platform-registration:$VERSION
# (4)
./demo/run-demo.sh
```

Then, use your browser to nativate to [localhost:3000](localhost:3000). User is `admin` and password is `admin`. You can skip the upgrade of password in this demonstration.

After login, click on dashboards on the left side menu and check the Registration Service dashboard out.
