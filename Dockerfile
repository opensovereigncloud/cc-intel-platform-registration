FROM ubuntu:22.04 AS builder

WORKDIR /root

RUN apt-get update && \
    env DEBIAN_FRONTEND=noninteractive apt-get install -y \
    wget \
    unzip \
    protobuf-compiler \
    libprotobuf-dev \
    build-essential \
    cmake \
    pkg-config \
    gdb \
    vim \
    python3 \
    git \
    gnupg && \
    apt-get -y -q upgrade && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# SGX SDK is installed in /opt/intel directory.
WORKDIR /opt/intel

ARG DCAP_VERSION=DCAP_1.21

RUN echo "deb [arch=amd64 signed-by=/usr/share/keyrings/intel-sgx.gpg] https://download.01.org/intel-sgx/sgx_repo/ubuntu jammy main" \
    | tee -a /etc/apt/sources.list.d/intel-sgx.list && \ 
    wget -qO - https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key | gpg --dearmor --output /usr/share/keyrings/intel-sgx.gpg && \
    apt-get update && \
    env DEBIAN_FRONTEND=noninteractive apt-get install -y \
    libsgx-dcap-ql-dev \
    libsgx-dcap-quote-verify-dev \
    libsgx-dcap-default-qpl-dev \
    libsgx-quote-ex-dev \
    libsgx-pce-logic

# Install SGX SDK
ARG SGX_SDK_URL=https://download.01.org/intel-sgx/sgx-linux/2.24/distro/ubuntu22.04-server/sgx_linux_x64_sdk_2.24.100.3.bin

RUN wget ${SGX_SDK_URL} && \
    export SGX_SDK_INSTALLER=$(basename $SGX_SDK_URL) && \
    chmod +x $SGX_SDK_INSTALLER && \
    echo "yes" | ./$SGX_SDK_INSTALLER && \
    rm $SGX_SDK_INSTALLER

ARG GO_MOD_VERSION

RUN wget https://golang.org/dl/go${GO_MOD_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_MOD_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_MOD_VERSION}.linux-amd64.tar.gz 

# Set Go environment variables
ENV PATH=$PATH:/usr/local/go/bin

ENV GOPATH=/go

ENV PATH=$PATH:$GOPATH/bin

WORKDIR /cc_build_dir

COPY . .

RUN make build

FROM ubuntu:22.04

RUN apt-get update && \
    apt-get install -y \
    wget \
    gnupg-agent

# Add 01.org to apt for SGX packages and install SGX runtime components
RUN echo "deb [arch=amd64 signed-by=/usr/share/keyrings/intel-sgx.gpg] https://download.01.org/intel-sgx/sgx_repo/ubuntu jammy main" \ 
    | tee -a /etc/apt/sources.list.d/intel-sgx.list && \
    wget -qO - https://download.01.org/intel-sgx/sgx_repo/ubuntu/intel-sgx-deb.key \ 
    | gpg --dearmor --output /usr/share/keyrings/intel-sgx.gpg && \
    apt-get update && \
    env DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    libsgx-enclave-common \
    libsgx-urts \
    libsgx-quote-ex \
    libsgx-ae-qve \
    libsgx-dcap-ql \
    libsgx-dcap-default-qpl \
    libsgx-pce-logic && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /cc_build_dir/build/lib/libsgx_platform_info.so /opt/cc-intel-platform-registration/
COPY --from=builder /cc_build_dir/build/lib/libmp_management.so /opt/cc-intel-platform-registration/
COPY --from=builder /cc_build_dir/build/lib/sgx_platform_enclave.signed.so /opt/cc-intel-platform-registration/

COPY --from=builder /cc_build_dir/cc-intel-platform-registration /usr/local/bin

ENV LD_LIBRARY_PATH=/opt/cc-intel-platform-registration

COPY docker-entrypoint.sh /usr/local/bin/

ENTRYPOINT ["docker-entrypoint.sh"]

CMD ["cc-intel-platform-registration"]
