set -x
ARGS=( -d analytics -c transactions --file /mnt/host/grafana-mongodb-community-plugin/integration-test/datasets/download/transactions.json )
if [ "${ALLOW_EMPTY_PASSWORD}" != "yes" ]; then
    ARGS+=(
        --username "${MONGODB_ROOT_USER}"
        --password "${MONGODB_ROOT_PASSWORD}"
        --authenticationDatabase admin
    )
fi
mongoimport \
    "${ARGS[@]}"
