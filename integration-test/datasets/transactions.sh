set -x
MONGODB_CLIENT_EXTRA_FLAGS=${MONGODB_CLIENT_EXTRA_FLAGS:-}
EXTRA_FLAGS="${MONGODB_CLIENT_EXTRA_FLAGS//--tls/--ssl}"
EXTRA_FLAGS="${EXTRA_FLAGS//sslCertificateKeyFile/sslPEMKeyFile}"
ARGS=( -d analytics -c transactions --file /mnt/host/grafana-mongodb-community-plugin/integration-test/datasets/download/transactions.json ${EXTRA_FLAGS} )
if [ "${ALLOW_EMPTY_PASSWORD}" != "yes" ]; then
    ARGS+=(
        --username "${MONGODB_ROOT_USER}"
        --password "${MONGODB_ROOT_PASSWORD}"
        --authenticationDatabase admin
    )
fi
mongoimport \
    "${ARGS[@]}"
