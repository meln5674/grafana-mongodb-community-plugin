docker run \
    -d \
    -p 3000:3000 \
    -e GF_INSTALL_PLUGINS="${ZIP_URL};meln5674-mongodb-community" \
    -e GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=meln5674-mongodb-community \
    grafana/grafana-oss:latest
