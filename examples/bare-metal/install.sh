#!/bin/bash -xeu

grafana-cli --pluginUrl "${ZIP_URL}" plugins install meln5674-mongodb-community
