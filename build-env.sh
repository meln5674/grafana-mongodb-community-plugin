#!/bin/bash -xeu

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
    -u "$(id -u):$(id -g)"
    -v "/${PWD}:/${PWD}"
    -v "/${HOME}:/${HOME}"
    -v /etc/passwd:/etc/passwd
    -v /etc/group:/etc/group
    -v /var/run/docker.sock:/var/run/docker.sock
    -e HOME
    -w "/${PWD}"
)

for group in $(id -G); do
    DOCKER_RUN_ARGS+=( --group-add "${group}" )
done

if [ -n "${GOPATH}" ]; then
    DOCKER_RUN_ARGS+=(
        -e GOPATH
        -v "/${GOPATH}:/${GOPATH}"
    )
fi

docker run "${DOCKER_RUN_ARGS[@]}" "${IMAGE}" "${COMMAND[@]}"
