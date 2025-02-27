#!/bin/bash

if [ -f /opt/intel/sgxsdk/environment ]; then
  source /opt/intel/sgxsdk/environment
fi

# The Intel libraries still expect the old /dev/sgx/* devices, even when the new in-kernel driver exposes
# /dev/sgx_enclave and /dev/sgx_provision. On the host, these devices are symlinked through udev rules distributed with
# the driver. In the container, on the other hand, we have to symlink them manually - unless they were already mapped in:
if [[ -e /dev/sgx_enclave && ! -e /dev/sgx/enclave ]]; then
    mkdir -p /dev/sgx
    ln -s /dev/sgx_enclave /dev/sgx/enclave
fi
if [[ -e /dev/sgx_provision && ! -e /dev/sgx/provision ]]; then
    mkdir -p /dev/sgx
    ln -s /dev/sgx_provision /dev/sgx/provision
fi

# AESM is usually a daemon running in the background and logging status messages to system's log daemon.
# We run it in a container instead and would like to see these log messages in the container's log.
# Therefore, they must be produced to the standard command line (stdout/stderr).
#
# AESM has a forground mode in which it writes its log messages to `/dev/console`.
# Thus, we run AESM in foreground mode (`--no-daemon` option).
#
# `/dev/console` is, by default, only available for interactive containers (`-i` option of `docker run`).
# Therefore, we link `/dev/console` to PID 1's stdout if the file doesn't exist, since PID 1's stdout ends up
# in the container's log.
if [[ ! -e /dev/console ]]; then
  ln -s /proc/1/fd/1 /dev/console
fi

# Apparently, the above is not sufficient to get all log messages

# Are we running in a TTY?
if ! tty -s; then
    echo "WARNING: Container runs without TTY. Azure DCAP QPL Log messages won't be visible."
    echo "Please run this container with TTY with the '-t' option in docker run, 'tty: true' in helm charts or docker-compose files."
fi

make build && ./cc-intel-platform-registration start



