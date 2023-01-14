# Sat Apr 25 16:19:01 +0000 2015
# Mon Jan 02 15:04:05 -0700 2006
# "RubyDate"
set -x
MONGODB_CLIENT_EXTRA_FLAGS=${MONGODB_CLIENT_EXTRA_FLAGS:-}
EXTRA_FLAGS="${MONGODB_CLIENT_EXTRA_FLAGS//--tls/--ssl}"
EXTRA_FLAGS="${EXTRA_FLAGS//sslCertificateKeyFile/sslPEMKeyFile}"
ARGS=( ${EXTRA_FLAGS} )
if [ "${ALLOW_EMPTY_PASSWORD}" != "yes" ]; then
    ARGS+=(
        --username "${MONGODB_ROOT_USER}"
        --password "${MONGODB_ROOT_PASSWORD}"
    )
fi
mongorestore \
    "${ARGS[@]}" \
    /mnt/host/grafana-mongodb-community-plugin/integration-test/datasets/download/tweets/dump/
