# Base image: Go devcontainer with Debian bookworm
FROM mcr.microsoft.com/devcontainers/go:1-1.24

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    bash-completion \
    git \
    zsh \
    iproute2 \
    iptables \
    socat \
    conntrack \
    && rm -rf /var/lib/apt/lists/*

# Install kubectl v1.34.1
RUN ARCH=$(dpkg --print-architecture) && \
    curl -LO "https://dl.k8s.io/release/v1.34.1/bin/linux/${ARCH}/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    rm kubectl

# Install kind (latest stable release)
RUN curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-$(dpkg --print-architecture) && \
    install -o root -g root -m 0755 kind /usr/local/bin/kind && \
    rm kind

# Add alias kc=kubectl + enable completion
RUN echo "alias kc='kubectl --insecure-skip-tls-verify=true'" >> /etc/bash.bashrc \
    && echo "source <(kubectl completion bash)" >> /etc/bash.bashrc \
    && echo "alias kc='kubectl --insecure-skip-tls-verify=true'" >> /etc/zsh/zshrc \
    && echo "source <(kubectl completion zsh)" >> /etc/zsh/zshrc

# Install ko

# Install ko (latest release)
RUN KO_VERSION=$(curl -s https://api.github.com/repos/ko-build/ko/releases/latest \
    | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/') && \
    curl -L https://github.com/ko-build/ko/releases/download/v${KO_VERSION}/ko_Linux_$(dpkg --print-architecture).tar.gz \
    | tar xz -C /usr/local/bin
