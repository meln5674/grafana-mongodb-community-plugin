docker run \
    -d \
    -p 3000:3000 \
    -e GF_INSTALL_PLUGINS="meln5674-mongodb-community=${ZIP_URL}" \
    -e GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=meln5674-mongodb-community \
    bitnami/grafana:latest
