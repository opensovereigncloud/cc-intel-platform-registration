#!/bin/bash
set -euo pipefail

# The Intel libraries still expect the old /dev/sgx/* devices, even when the new in-kernel driver exposes
# /dev/sgx_enclave and /dev/sgx_provision. On the host, these devices are symlinked through udev rules distributed with
# the driver. In the container, on the other hand, we have to symlink them manually - unless they were already mapped in:
if [[ -e /dev/sgx_enclave && ! -e /dev/sgx/enclave ]]; then
  mkdir -p /dev/sgx
  ln -s /dev/sgx_enclave /dev/sgx/enclave
fi

if [ "$1" = 'cc-intel-platform-registration' ]; then

  exec cc-intel-platform-registration "$@"
fi

exec "$@"