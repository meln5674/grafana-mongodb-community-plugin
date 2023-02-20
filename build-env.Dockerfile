ARG BASE_IMAGE=docker.io/library/debian:9
ARG GO_VERSION=1.18.4
ARG GO_IMAGE=docker.io/library/golang:${GO_VERSION}
ARG DOCKER_VERSION=20.10.23
ARG DOCKER_IMAGE=docker.io/library/docker:${DOCKER_VERSION}-cli
ARG ARCH=amd64
ARG NODE_ARCH=x64
ARG OS=linux

FROM ${BASE_IMAGE} AS base

RUN apt-get update && apt-get install -y curl xz-utils 

ENV PATH=${PATH}:/usr/local/lib/nodejs/bin:/usr/local/go/bin:/go/bin
ENV GOPATH=/go

FROM base AS kubectl

ARG KUBECTL_VERSION=v1.22.0

ARG ARCH
ARG OS

ADD "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${OS}/${ARCH}/kubectl" /usr/local/bin/

RUN chmod +x /usr/local/bin/kubectl

FROM base AS node

ARG NODE_VERSION=v14.20.0

ARG NODE_ARCH
ARG OS

RUN mkdir -p /usr/local/lib/nodejs \
 && curl -vfL https://nodejs.org/dist/${NODE_VERSION}/node-${NODE_VERSION}-${OS}-${NODE_ARCH}.tar.xz | tar -xJ --strip-components=1 -C /usr/local/lib/nodejs \
 && npm install -g yarn

FROM ${GO_IMAGE} AS go

RUN go install github.com/magefile/mage@v1.13.0

FROM ${DOCKER_IMAGE} AS docker

FROM base AS helm

ARG ARCH
ARG OS
ARG HELM_VERSION=v3.10.3

ADD https://get.helm.sh/helm-${HELM_VERSION}-${OS}-${ARCH}.tar.gz /usr/local/bin/

RUN tar -xzf /usr/local/bin/helm-${HELM_VERSION}-${OS}-${ARCH}.tar.gz -C /usr/local/bin

FROM base AS kind

ARG ARCH
ARG OS
ARG KIND_VERSION=v0.14.0

ADD https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-${OS}-${ARCH} /usr/local/bin/

RUN chmod +x /usr/local/bin/kind-${OS}-${ARCH}

FROM base

ARG ARCH
ARG OS

RUN apt-get install -y git gcc g++ zip unzip make

COPY --from=kubectl /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
COPY --from=kind /usr/local/bin/kind-${OS}-${ARCH} /usr/local/bin/kind
COPY --from=helm /usr/local/bin/${OS}-${ARCH}/helm /usr/local/bin/helm
COPY --from=go /usr/local/go /usr/local/go
COPY --from=go /go/bin/mage /usr/local/bin/mage
COPY --from=node /usr/local/lib/nodejs /usr/local/lib/nodejs

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
 
ENV GOPATH=/go
VOLUME /go
