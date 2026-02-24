#!/usr/bin/env bash

set -e

# Configuration
IMAGE_NAME="cc-intel-platform-registration-dev"
VERSION="dev"
GO_MOD_VERSION="1.26.0"
CONTAINER_NAME="cc-intel-platform-registration-dev"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${GREEN}Building development image (builder target)...${NC}"
docker build \
    --target builder \
    --build-arg GO_MOD_VERSION=${GO_MOD_VERSION} \
    --build-arg TARGET_VERSION=${VERSION} \
    -t ${IMAGE_NAME}:${VERSION} \
    "${SCRIPT_DIR}"

echo -e "${GREEN}Starting development container...${NC}"

# Check if container already exists
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${YELLOW}Container ${CONTAINER_NAME} already exists. Removing it...${NC}"
    docker rm -f ${CONTAINER_NAME}
fi

# Start the container
docker run -it --rm \
    --name ${CONTAINER_NAME} \
    --privileged \
    --device /dev/sgx_enclave:/dev/sgx_enclave \
    --device /dev/sgx_provision:/dev/sgx_provision \
    -v "${SCRIPT_DIR}:/cc_build_dir" \
    -v /sys/firmware/efi/efivars:/sys/firmware/efi/efivars \
    -e LD_LIBRARY_PATH="$LD_LIBRARY_PATH:/cc_build_dir/build/lib" \
    -w /cc_build_dir \
    ${IMAGE_NAME}:${VERSION} \
    /bin/bash -c "make clean && /bin/bash"

echo -e "${GREEN}Development container stopped.${NC}"
