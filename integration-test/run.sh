#!/bin/bash -xe

# Create a KinD cluster with the repo checkout mounted
# Deploy an NGinx server with Helm that mounts the current directory at the serving root
# Deploy a MongoDB server with Helm
# Deploy A Grafana server with Helm that allows loading the unsigned plugin and has a pre-configured datasource for it
#   If in dev mode, mount the current directory at the correct plugin location
#   If not in dev mode, set the plugins env var to download the plugin zip from NGinx

# Once grafana is installed
    # If in dev mode, restart grafana to ensure any changes take effect, then start port forwarding
    # If not in dev mode, wait for grafana to become health in a reasonable amount of time, then run a Job
    #     that hits the datasource

export KUBECONFIG=${KUBECONFIG:-integration-test/kubeconfig}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-meln5674-mongodb-community-it}
if ! kind get clusters | grep -q "${KIND_CLUSTER_NAME}" ; then
    sed "s/hostPath: .*/hostPath: '${PWD//\//\\/}'/" < integration-test/kind.config.template > integration-test/kind.config

    kind create cluster --name "${KIND_CLUSTER_NAME}" --kubeconfig "${KUBECONFIG}" --config integration-test/kind.config
    if [ -z "${INTEGRATION_TEST_NO_CLEANUP}" ]; then
        trap "kind delete cluster --name '${KIND_CLUSTER_NAME}'" EXIT
    fi
fi

kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: mongodb-init
data:
  weather.js: |
$(set +x; while IFS= read -r line; do echo "    ${line}" ; done < integration-test/weather.js; set +x)
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-ini
data:
  grafana.ini: |
$(set +x; while IFS= read -r line; do echo "    ${line}" ; done < integration-test/grafana.ini; set -x)
---
apiVersion: v1
kind: Secret
metadata:
  name: datasources
stringData:
  datasources.yaml: |
$(set +x; while IFS= read -r line; do echo "    ${line}" ; done < integration-test/datasources.yaml; set -x)
EOF

helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

NGINX_ARGS=(
    --set fullnameOverride=plugin-repo
    --set extraVolumes[0].name=plugin
    --set extraVolumes[0].hostPath.path=/mnt/host/grafana-mongodb-community-plugin/
    --set extraVolumeMounts[0].name=plugin
    --set extraVolumeMounts[0].mountPath=/opt/bitnami/nginx/html/grafana-mongodb-community-plugin/
)

helm upgrade --install plugin-repo bitnami/nginx "${NGINX_ARGS[@]}"

MONGODB_ARGS=(
    --version 12.1.26
    --set auth.rootPassword=root
    --set initdbScriptsConfigMap=mongodb-init
    --set useStatefulSet=true
)

helm upgrade --install mongodb bitnami/mongodb "${MONGODB_ARGS[@]}"

GRAFANA_ARGS=(
    --set datasources.secretName=datasources
    --set admin.password=admin
    --set config.grafanaIniConfigMap=grafana-ini
    --set config.useGrafanaIniFile=true
    # --set image.tag=7.1.5-debian-10-r9
)

if [ -n "${INTEGRATION_TEST_DEV_MODE}" ]; then
    GRAFANA_ARGS+=(
        --set grafana.extraVolumes[0].name=plugin
        --set grafana.extraVolumes[0].hostPath.path=/mnt/host/grafana-mongodb-community-plugin/
        --set grafana.extraVolumeMounts[0].name=plugin
        --set grafana.extraVolumeMounts[0].mountPath=/opt/bitnami/grafana/data/plugins/meln5674-mongodb-community
    )
else
    GRAFANA_ARGS+=( 
        --set plugins='meln5674-mongodb-community=http://plugin-repo/grafana-mongodb-community-plugin/meln5674-mongodb-community.zip'
    )
fi



helm upgrade --install grafana bitnami/grafana "${GRAFANA_ARGS[@]}"

        
if [ -n "${INTEGRATION_TEST_DEV_MODE}" ]; then
    kubectl rollout restart deploy/grafana
    kubectl rollout status deploy/grafana
    sleep 2
    echo 'Forwarding ports. Press Ctrl+C to exit and re-run this script to make changes'
    kubectl port-forward deploy/grafana 3000:3000
else
    kubectl wait deploy/grafana --for=condition=available --timeout=120s
    kubectl replace --force -f - <<EOF

apiVersion: batch/v1
kind: Job
metadata:
  name: grafana-mongodb-community-plugin-it
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: curl
        image: docker.io/alpine/curl:latest
        command: [sh, -exuc]
        args:
        - |
            curl -v -f -u admin:admin http://grafana:3000/api/datasources/1/health
            # [{ "$project": { "timestamp": 1, "sensorID": "$metadata.sensorId", "temperature": "$temp", "foo": { "$literal": ${__from} } }}]
            curl 'http://grafana:3000/api/ds/query' \
              -v -f \
              -u admin:admin \
              -H 'accept: application/json, text/plain, */*' \
              -H 'content-type: application/json' \
              --data-raw '$(cat integration-test/query.json)'
EOF


    kubectl wait job/grafana-mongodb-community-plugin-it --for=condition=complete &
    kubectl wait job/grafana-mongodb-community-plugin-it --for=condition=failed &

    wait -n

    kill $(jobs -p)

    if ! kubectl wait job/grafana-mongodb-community-plugin-it --for=condition=complete --timeout=0; then
        kubectl logs job/grafana-mongodb-community-plugin-it

        echo
        exit 1
    fi
fi
