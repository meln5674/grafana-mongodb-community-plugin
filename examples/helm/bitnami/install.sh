#!/bin/bash -xeu

helm repo add bitnami https://charts.bitnami.com/bitnami
helm upgrade \
    --wait --install --debug \
    grafana bitnami/grafana \
    --set plugins="meln5674-mongodb-community=${ZIP_URL}"
