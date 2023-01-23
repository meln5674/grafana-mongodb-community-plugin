ARG BASE_IMAGE=docker.io/library/debian:9

FROM ${BASE_IMAGE} AS base

RUN apt-get update && apt-get install -y curl xz-utils git gcc g++ zip unzip make

ENV PATH=${PATH}:/usr/local/lib/nodejs/bin:/usr/local/go/bin:/gopath/bin
ENV GOPATH=/gopath

FROM base AS kubectl

ARG KUBECTL_VERSION=v1.22.0

RUN curl -vfL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" > /usr/local/bin/kubectl \
 && chmod +x /usr/local/bin/kubectl

FROM base AS node

ARG NODE_VERSION=v14.20.0

RUN mkdir -p /usr/local/lib/nodejs \
 && curl -vfL https://nodejs.org/dist/${NODE_VERSION}/node-${NODE_VERSION}-linux-x64.tar.xz | tar -xJ --strip-components=1 -C /usr/local/lib/nodejs \
 && npm install -g yarn @grafana/toolkit

FROM base AS go

RUN mkdir -p /usr/local/go \
 && curl -vfL https://go.dev/dl/go1.18.4.linux-amd64.tar.gz | tar -xz -C /usr/local/

RUN go install github.com/magefile/mage@v1.13.0

FROM base AS docker

ARG DOCKER_VERSION=20.10.9

RUN curl -vfL https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}.tgz | tar xz --strip-components=1 -C /usr/local/bin docker/docker

FROM base AS helm

ARG HELM_VERSION=v3.10.3

RUN curl -vfL https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar xz --strip-components=1 -C /usr/local/bin linux-amd64/helm

FROM base AS kind

ARG KIND_VERSION=v0.14.0

RUN curl -vfL https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-linux-amd64 > /usr/local/bin/kind \
 && chmod +x /usr/local/bin/kind

FROM base

RUN 

COPY --from=kubectl /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=node /usr/local/lib/nodejs /usr/local/lib/nodejs
COPY --from=go /usr/local/go /usr/local/go
COPY --from=go ${GOPATH} ${GOPATH}
COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
COPY --from=kind /usr/local/bin/kind /usr/local/bin/kind
COPY --from=helm /usr/local/bin/helm /usr/local/bin/helm

RUN node --version \
 && npm --version \
 && yarn --version \
 && go version \
 && mage --help \
 && zip --version \
 && gcc --version \
 && g++ --version \
 && make --version \
 && helm version \
 && unzip -h \
 && grafana-toolkit --help
 
ENV GOPATH=/go
VOLUME /go
