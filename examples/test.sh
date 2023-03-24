#!/bin/bash -xeu

# NOTE: This file is for testing the examples, you should use the example specific to your use-case

function retry_curl {
    RETRIES=$1
    POLL_PERIOD=$2
    for retry in $(seq "${RETRIES}"); do 
        if curl -vL localhost:3000; then
            break
        fi
        sleep "${POLL_PERIOD}"
    done
    curl -vL localhost:3000
}

function defer {
    CMD=$1
    TRAP_CMD="${CMD} ; ${TRAP_CMD:-}"
    trap "set +e ; ${TRAP_CMD}" EXIT
}

VOLUME_DIR=$(mktemp -d)
VOLUME_PARENT_DIR=$(dirname "${VOLUME_DIR}")
defer "docker run --rm -i -v '${VOLUME_PARENT_DIR}:${VOLUME_PARENT_DIR}' alpine rm -rf '${VOLUME_DIR}'"

#
# bare-metal
#
docker run \
    --rm -i \
    -v "${VOLUME_DIR}:/volumes" \
    alpine sh -xe \
<<EOF
mkdir -p /volumes/bare-metal /volumes/docker/bitnami /volumes/docker/official /volumes/docker-compose/bitnami /volumes/docker-compose/official
chown 472:$(id -g) /volumes/*
chmod 0770 /volumes/*
EOF

docker pull grafana/grafana-oss:latest
docker pull bitnami/grafana:latest

docker run \
    --name meln5674-grafana-mongodb \
    --rm -d \
    -e ZIP_URL \
    -p 3000:3000 \
    -v "${VOLUME_DIR}/bare-metal:/var/lib/grafana" \
    --entrypoint bash \
    grafana/grafana-oss:latest \
    -c 'tail -f /dev/null'
defer "docker kill meln5674-grafana-mongodb"

docker cp bare-metal/grafana.ini meln5674-grafana-mongodb:/etc/grafana/grafana.ini
docker cp bare-metal/install.sh meln5674-grafana-mongodb:/install.sh
(
docker exec -i meln5674-grafana-mongodb bash -xe \
<<EOF
    /install.sh
    /run.sh
EOF
) &
INSTALL_PID="$!"
defer "kill '${INSTALL_PID}'"

retry_curl 10 15

docker kill meln5674-grafana-mongodb

#
# docker/official
#
CONTAINER_ID=$(./docker/official/install.sh)
defer "docker kill '${CONTAINER_ID}'; docker rm '${CONTAINER_ID}'"
docker logs -f "${CONTAINER_ID}" &

retry_curl 10 15


docker kill "${CONTAINER_ID}"
docker rm "${CONTAINER_ID}"

#
# docker/bitnami
#
CONTAINER_ID=$(./docker/bitnami/install.sh)
defer "docker kill '${CONTAINER_ID}'; docker rm '${CONTAINER_ID}'"
docker logs -f "${CONTAINER_ID}" &

retry_curl 10 15

docker kill "${CONTAINER_ID}"
docker rm "${CONTAINER_ID}"

#
# docker-compose/official
#
docker-compose --project-directory docker-compose/official up -d
docker-compose --project-directory docker-compose/official logs -f &
defer "docker-compose --project-directory docker-compose/official down"

retry_curl 10 15

docker-compose --project-directory docker-compose/official down

#
# docker-compose/bitnami
#
docker-compose --project-directory docker-compose/bitnami up -d
docker-compose --project-directory docker-compose/bitnami logs -f &
defer "docker-compose --project-directory docker-compose/bitnami down"

retry_curl 10 15
docker-compose --project-directory docker-compose/bitnami down


kind create cluster --name=meln5674-grafana-mongodb --kubeconfig=test.kubeconfig
kind load docker-image --name=meln5674-grafana-mongodb grafana/grafana-oss:latest bitnami/grafana:latest
export KUBECONFIG=test.kubeconfig
defer 'kind delete cluster --name=meln5674-grafana-mongodb; rm test.kubeconfig'

#
# kubectl/grafana
#
cat kubectl/official/deployment.yaml | envsubst | kubectl create -f -
defer 'kubectl delete -f kubectl/official/'
kubectl wait --for=condition=Available -f kubectl/official/deployment.yaml --timeout=120s
kubectl port-forward deploy/grafana 3000:3000 &
PORT_FORWARD_PID=$!
defer "kill '${PORT_FORWARD_PID}'"

retry_curl 10 15
kill "${PORT_FORWARD_PID}"
while [ -n "$(jobs)" ]; do
    wait
done
kubectl delete -f kubectl/official/

#
# kubectl/bitnami
#
cat kubectl/bitnami/deployment.yaml | envsubst | kubectl create -f -
defer 'kubectl delete -f kubectl/bitnami/'
kubectl wait --for=condition=Available -f kubectl/bitnami/deployment.yaml --timeout=120s
kubectl port-forward deploy/grafana 3000:3000 &
PORT_FORWARD_PID=$!
defer "kill '${PORT_FORWARD_PID}'"

retry_curl 10 15
kill "${PORT_FORWARD_PID}"
while [ -n "$(jobs)" ]; do
    wait
done
kubectl delete -f kubectl/bitnami/

#
# helm/bitnami
#
helm/bitnami/install.sh
defer 'helm delete grafana'
kubectl wait --for=condition=Available deploy/grafana --timeout=120s
kubectl port-forward deploy/grafana 3000:3000 &
PORT_FORWARD_PID=$!
defer "kill '${PORT_FORWARD_PID}'"

retry_curl 10 15
kill "${PORT_FORWARD_PID}"
while [ -n "$(jobs)" ]; do
    wait
done
helm delete grafana

kind delete cluster --name=meln5674-grafana-mongodb




echo 'All Tests Passed!'
