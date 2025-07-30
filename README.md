# CC Intel Platform Registration

This repository contains the `cc-intel-platform-registration` service, which automates the registration of multi-socket SGX with the Intel SGX Registration service.
It exposes metrics that can be visualized through Prometheus and Grafana to monitor the registration process and status.

![Visualization of the CC-IPCEIS](/ipceis_diagram.jpg)

## Metrics

The service exposes the following metrics via Prometheus:

- Registration status (`service_status_code`): Current status code of the registration service.
- Registration Service Panic Counts (`application_panics_total`): Total number of go routines panics.

These metrics can be visualized through a Grafana dashboard to monitor the platform registration process.

## Prerequisites

- Helm (for Kubernetes deployment)
- Docker and docker-compose (for local deployment)
- EFI System Variables: `cc-intel-platform-registration` needs `read` and `write` capabilities to the host device SGX UEFI variables. 
It must be able to mount the `/sys/firmware/efi/efivars` host path.

    When creating your cluster with `k3d`, you should explicitly bind the `/sys/firmware/efi/efivars` volume to your nodes:

    ```bash
    # Example: A cluster with 1 server and 2 agent nodes
    k3d cluster create my-cluster-name \
        --agents 2 \
        -v /sys/firmware/efi/efivars:/sys/firmware/efi/efivars@agent:0,1 \
        -v /sys/firmware/efi/efivars:/sys/firmware/efi/efivars@server:0 
    ```
- SGX Device Support: The service requires a `sgx.intel.com/enclave: 1` resource on Kubernetes. This is typically provided by a SGX plugin Daemonset. 
For deploying with docker-compose, it requires access to `/dev/sgx_enclave` provided by the host machine.

## Installation

### Get the latest Image

```bash
docker pull ghcr.io/opensovereigncloud/cc-intel-platform-registration:latest
```

### Or build the Image

If you want to build the image yourself:

```bash
# Set environment variables
export REGISTRY=my_registry_name

export VERSION=latest

# Build the image
make build-image IMAGE_REGISTRY=$REGISTRY VERSION=$VERSION

# Push to registry (optional)
docker push $REGISTRY/cc-intel-platform-registration:$VERSION
```

This repository contains [helm charts](/charts) for easy installation on Kubernetes.

### Running the Demo script

The fastest way to setup is by running the demo script. This would setup grafana and prometheus, and deploy the service with Helm or docker compose.

1. Clone this repository and navigate to the root directory:

    ```bash
    git clone https://github.com/opensovereigncloud/cc-intel-platform-registration.git

    cd cc-intel-platform-registration
   ```

1. Configure environment variables:

    ```bash
    cp .env.template .env

    # Edit the environmental variables as needed
    nano .env

    source .env
   ```

1. Run the demo script:

    ```bash
    chmod +x ./demo/run-demo.sh

    ./demo/run-demo.sh
   ```

1. Choose your deployment option when prompted (Kubernetes or docker-compose).

### Accessing the Grafana Dashboard

After the demo deployment:

1. Navigate to Grafana in your browser:
    - For local deployment: [http://localhost:3000](http://localhost:3000)
    - For Kubernetes deployment: Use port-forwarding or an ingress if configured

1. Log in with the default credentials:
    - Username: `admin`
    - Password: `admin`

   You can skip the password change for demo purposes.
1. Click on "Dashboards" in the left side menu and select the "Registration Service" dashboard.
