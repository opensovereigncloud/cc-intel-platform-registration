FROM ubuntu:22.04

COPY sgx-files.sha256 /tmp

# package tzdata attempts to start an interactive configuration by default, disable this:
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y wget libssl-dev libcurl4-openssl-dev libprotobuf-dev build-essential cmake libxml2-dev uuid-dev efivar efibootmgr && \
    mkdir download && \
    cd download && \
    wget https://download.01.org/intel-sgx/sgx-linux/2.24/distro/ubuntu22.04-server/sgx_debian_local_repo.tgz && \
    wget https://download.01.org/intel-sgx/sgx-linux/2.24/distro/ubuntu22.04-server/sgx_linux_x64_sdk_2.24.100.3.bin && \
    sha256sum -c /tmp/sgx-files.sha256 && \
    mkdir -p /etc/init && \
    tar -xf sgx_debian_local_repo.tgz && \
    cd sgx_debian_local_repo/pool/main && \
    dpkg -i libs/libsgx-ae-id-enclave/libsgx-ae-id-enclave_1.21.100.3-jammy1_amd64.deb \
            libs/libsgx-ae-qe3/libsgx-ae-qe3_1.21.100.3-jammy1_amd64.deb \
            libs/libsgx-headers/libsgx-headers_2.24.100.3-jammy1_amd64.deb \
            libs/libsgx-dcap-ql/libsgx-dcap-ql_1.21.100.3-jammy1_amd64.deb \
            libs/libsgx-dcap-ql/libsgx-dcap-ql-dev_1.21.100.3-jammy1_amd64.deb \
            libs/libsgx-dcap-default-qpl/libsgx-dcap-default-qpl_1.21.100.3-jammy1_amd64.deb \
            libs/libsgx-dcap-default-qpl/libsgx-dcap-default-qpl-dev_1.21.100.3-jammy1_amd64.deb \
            libs/libsgx-qe3-logic/libsgx-qe3-logic_1.21.100.3-jammy1_amd64.deb \
            libs/libsgx-pce-logic/libsgx-pce-logic_1.21.100.3-jammy1_amd64.deb \
            libs/libsgx-enclave-common/libsgx-enclave-common_2.24.100.3-jammy1_amd64.deb \
            libs/libsgx-enclave-common/libsgx-enclave-common-dev_2.24.100.3-jammy1_amd64.deb \
            libs/libsgx-epid/libsgx-epid_2.24.100.3-jammy1_amd64.deb \
            libs/libsgx-launch/libsgx-launch_2.24.100.3-jammy1_amd64.deb \
            libs/libsgx-quote-ex/libsgx-quote-ex_2.24.100.3-jammy1_amd64.deb \
            libs/libsgx-quote-ex/libsgx-quote-ex-dev_2.24.100.3-jammy1_amd64.deb \
            libs/libsgx-uae-service/libsgx-uae-service_2.24.100.3-jammy1_amd64.deb \
            libs/libsgx-urts/libsgx-urts_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/libsgx-ae-epid_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/libsgx-ae-le_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/libsgx-ae-pce_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/libsgx-aesm-ecdsa-plugin_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/libsgx-aesm-epid-plugin_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/libsgx-aesm-launch-plugin_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/libsgx-aesm-pce-plugin_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/libsgx-aesm-quote-ex-plugin_2.24.100.3-jammy1_amd64.deb \
            s/sgx-aesm-service/sgx-aesm-service_2.24.100.3-jammy1_amd64.deb && \
    cd - && \
    chmod +x ./sgx_linux_x64_sdk_2.24.100.3.bin && \
    ./sgx_linux_x64_sdk_2.24.100.3.bin --prefix=/opt/intel

ENV GO_VERSION=1.22.9
RUN wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

# Set Go environment variables
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/go
ENV PATH=$PATH:$GOPATH/bin

COPY entrypoint.sh /entrypoint.sh

WORKDIR /usr/src/cc-intel-platform-registration

COPY . .

ENTRYPOINT ["/entrypoint.sh"]
