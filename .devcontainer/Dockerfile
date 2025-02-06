FROM mcr.microsoft.com/devcontainers/javascript-node@896bfba10582c9239d1c36bab53b80af06253019f62b846fa440ee643ca63eb1
# 22-bookworm

ARG GO_VERSION="1.23.6"
ARG GO_CHECKSUM="9379441ea310de000f33a4dc767bd966e72ab2826270e038e78b2c53c2e7802d"
ARG KUBECTL_VERSION="1.31.0"

# gcloud-cli
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
    && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - \
    && apt-get update -y && apt-get install google-cloud-cli -y \
    && rm -rf /var/lib/apt/lists/*

# kubectl
RUN curl -sLO "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl" \
    && curl -sLO "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256" \
    && echo "$(cat kubectl.sha256)  kubectl" | sha256sum --check \
    && install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl \
    && rm kubectl kubectl.sha256

RUN echo 'source <(kubectl completion bash)' >>~/.bashrc 

# golang
RUN curl -sLO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
    && echo "${GO_CHECKSUM}  go${GO_VERSION}.linux-amd64.tar.gz" | sha256sum --check \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH $PATH:/usr/local/go/bin

VOLUME [ "web/node_modules" ]
