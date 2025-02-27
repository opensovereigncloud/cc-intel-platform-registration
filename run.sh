#!/bin/bash
docker build -t cc-intel-platform-registration:latest .

docker run --privileged  -it --rm  --volume /sys/firmware/efi/efivars:/sys/firmware/efi/efivars  --device=/dev/sgx_enclave:/dev/sgx_enclave \
    --device=/dev/sgx_provision:/dev/sgx_provision  cc-intel-platform-registration
 