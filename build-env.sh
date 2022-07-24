#!/bin/bash -xeu

# Assume the user wants an interactive shell if no args are passed, otherwise, run the provided command
if [ "$#" -eq 0 ]; then
    COMMAND=( bash )
else
    COMMAND=( "$@" )
fi

export DOCKER_BUILDKIT=1

IMAGE_REPO=localhost/meln5674/grafana-mongodb-community-plugin

IMAGE_TAG=$(md5sum build-env.Dockerfile | awk '{ print $1 }')

IMAGE="${IMAGE_REPO}/${IMAGE_TAG}"

docker build -f build-env.Dockerfile -t "${IMAGE}" .


DOCKER_RUN_ARGS=(
    --rm
    -it
)

# Make it look like we're in the same directory as we ran from
DOCKER_RUN_ARGS+=(
    -v "/${PWD}:/${PWD}"
    -w "/${PWD}"
)

# Make it look like we're the same user
DOCKER_RUN_ARGS+=(
    -u "$(id -u):$(id -g)"
    -v "/${HOME}:/${HOME}"
    -v /etc/passwd:/etc/passwd
    -v /etc/group:/etc/group
    -e HOME
)
for group in $(id -G); do
    DOCKER_RUN_ARGS+=( --group-add "${group}" )
done

# Provide access to docker
DOCKER_RUN_ARGS+=(
    -v /var/run/docker.sock:/var/run/docker.sock
)

# Provide access to an existing kind cluster, as well as enable port-forwarding for live dev env
DOCKER_RUN_ARGS+=(
    -e KUBECONFIG
    --network host
)

# If GOPATH is set, also mount it and forward the env so we can re-use the package cache
if [ -n "${GOPATH}" ]; then
    DOCKER_RUN_ARGS+=(
        -e GOPATH
        -v "/${GOPATH}:/${GOPATH}"
    )
fi

# Forward variables used by the integration test scripts
DOCKER_RUN_ARGS+=(
    -e INTEGRATION_TEST_DEV_MODE
    -e INTEGRATION_TEST_NO_CLEANUP
    -e KIND_CLUSTER_NAME
)

exec docker run "${DOCKER_RUN_ARGS[@]}" "${IMAGE}" "${COMMAND[@]}"
