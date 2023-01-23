#!/bin/bash -xeu

# Ensure datasets are downloaded
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

# Read a file and print it as a JSON string, i.e. with ""'s and \n's    
function as-json {
    echo -n '"'
    while read -r line ; do
        echo -n "${line//'"'/'\"'}"'\n'
    done
    echo -n '"'
}

if [ -z "${INTEGRATION_TEST_ONLY_TESTS:-}" ]; then

if ! [ -f integration-test/datasets/download/tweets.zip ]; then
    curl -vfL https://github.com/ozlerhakan/mongodb-json-files/blob/master/datasets/tweets.zip?raw=true > integration-test/datasets/download/tweets.zip
fi
if ! [ -f integration-test/datasets/download/transactions.json ]; then
    curl -vfL https://github.com/fieldsets/mongodb-sample-datasets/blob/main/sample_analytics/transactions.json?raw=true > integration-test/datasets/download/transactions.json
fi
rm -rf integration-test/datasets/download/tweets
unzip integration-test/datasets/download/tweets.zip -d integration-test/datasets/download/tweets/

export KUBECONFIG=${KUBECONFIG:-integration-test/kubeconfig}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-meln5674-mongodb-community-it}
if ! kind get clusters | grep -q "${KIND_CLUSTER_NAME}" ; then
    sed "s/hostPath: .*/hostPath: '${PWD//\//\\/}'/" < integration-test/kind.config.template > integration-test/kind.config

    kind create cluster --name "${KIND_CLUSTER_NAME}" --kubeconfig "${KUBECONFIG}" --config integration-test/kind.config
    if [ -z "${INTEGRATION_TEST_NO_CLEANUP:-}" ]; then
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
$(set +x; while IFS= read -r line; do echo "    ${line}" ; done < integration-test/datasets/weather.js; set +x)
  tweets.sh: |
$(set +x; while IFS= read -r line; do echo "    ${line}" ; done < integration-test/datasets/tweets.sh; set +x)
  transations.sh: |
$(set +x; while IFS= read -r line; do echo "    ${line}" ; done < integration-test/datasets/transactions.sh; set +x)
---
EOF

helm repo add jetstack https://charts.jetstack.io
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

NGINX_ARGS=(
    --set fullnameOverride=plugin-repo
    --set extraVolumes[0].name=plugin
    --set extraVolumes[0].hostPath.path=/mnt/host/grafana-mongodb-community-plugin/
    --set extraVolumeMounts[0].name=plugin
    --set extraVolumeMounts[0].mountPath=/opt/bitnami/nginx/html/grafana-mongodb-community-plugin/
    --set service.type=ClusterIP
)

helm upgrade --install --wait plugin-repo bitnami/nginx "${NGINX_ARGS[@]}"

CERT_MANAGER_ARGS=(
    --set installCRDs=true
)

helm upgrade --install --wait cert-manager jetstack/cert-manager "${CERT_MANAGER_ARGS[@]}"

kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: mongodb-tls-ca
spec:
  isCA: true
  commonName: mongodb-tls-ca
  secretName: mongodb-tls-ca
  privateKey:
    algorithm: RSA
    size: 2048
  issuerRef:
    name: selfsigned
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: mongodb-tls-ca
spec:
  ca:
    secretName: mongodb-tls-ca
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: mongodb-mtls-client
spec:
  isCA: true
  commonName: mongodb-mtls-client
  secretName: mongodb-mtls-client
  privateKey:
    algorithm: RSA
    size: 2048
  issuerRef:
    name: mongodb-tls-ca
    kind: Issuer
    group: cert-manager.io
EOF

kubectl wait certificate/mongodb-tls-ca --for=condition=ready
kubectl wait certificate/mongodb-mtls-client --for=condition=ready

kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: mongodb-tls-certs
data:
    mongodb-ca-cert: $(kubectl get secret mongodb-tls-ca --template '{{ index .data "tls.crt" }}')
    mongodb-ca-key: $(kubectl get secret mongodb-tls-ca --template '{{ index .data "tls.key" }}')
EOF

MONGODB_ARGS=(
    --version 13.6.2
    --set auth.rootPassword=rootPassword
    --set initdbScriptsConfigMap=mongodb-init
    --set useStatefulSet=true
    --set extraVolumes[0].name=sample-data
    --set extraVolumes[0].hostPath.path=/mnt/host/grafana-mongodb-community-plugin/integration-test/datasets/download
    --set extraVolumeMounts[0].name=sample-data
    --set extraVolumeMounts[0].mountPath=/mnt/host/grafana-mongodb-community-plugin/integration-test/datasets/download
    --set image.debug=true
)

helm upgrade --install --wait mongodb bitnami/mongodb "${MONGODB_ARGS[@]}"

MONGODB_NOAUTH_ARGS+=( "${MONGODB_ARGS[@]}" --set auth.enabled=false )

helm upgrade --install --wait mongodb-no-auth bitnami/mongodb "${MONGODB_NOAUTH_ARGS[@]}"

MONGODB_TLS_CHART_REPO_PATH=integration-test/vendor/github.com/bitnami/charts
# See https://github.com/bitnami/charts/issues/13317
if [ -e "${MONGODB_TLS_CHART_REPO_PATH}" ]; then
    rm -rf "${MONGODB_TLS_CHART_REPO_PATH}"
fi
mkdir -p "${MONGODB_TLS_CHART_REPO_PATH}"
git clone https://github.com/meln5674/bitnami-charts.git "${MONGODB_TLS_CHART_REPO_PATH}"
git -C "${MONGODB_TLS_CHART_REPO_PATH}" pull --force
git -C "${MONGODB_TLS_CHART_REPO_PATH}" checkout feature/mongodb-tls-only-13317
MONGODB_TLS_CHART_PATH="${MONGODB_TLS_CHART_REPO_PATH}/bitnami/mongodb"
helm dep update "${MONGODB_TLS_CHART_PATH}"

MONGODB_MTLS_ARGS=(
    "${MONGODB_ARGS[@]}"
    --set tls.enabled=true
    --set tls.existingSecret=mongodb-tls-certs
)

helm upgrade --install --wait mongodb-mtls "${MONGODB_TLS_CHART_PATH}" "${MONGODB_MTLS_ARGS[@]}"

MONGODB_TLS_ARGS=(
    "${MONGODB_ARGS[@]}"
    --set tls.enabled=true
    --set tls.existingSecret=mongodb-tls-certs
    --set tls.mTLS.enabled=false
)



helm upgrade --install --wait mongodb-tls "${MONGODB_TLS_CHART_PATH}" "${MONGODB_TLS_ARGS[@]}"


MONGODB_CA=$(kubectl get secret mongodb-tls-certs --template '{{ index .data "mongodb-ca-cert" }}' | base64 -d | as-json)
MONGODB_CLIENT_CERT=$(kubectl get secret mongodb-mtls-client --template '{{ index .data "tls.crt" }}' | base64 -d | as-json)
MONGODB_CLIENT_KEY=$(kubectl get secret mongodb-mtls-client --template '{{ index .data "tls.key" }}' | base64 -d | as-json)

kubectl apply -f - <<EOF
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
$(
  set +x
  cat integration-test/datasources.yaml \
  | while IFS= read -r line; do
        echo "    ${line}"
    done \
  | sed "s|TLS_CERTIFICATE|${MONGODB_CLIENT_CERT//\\/\\\\}|" \
  | sed "s|TLS_KEY|${MONGODB_CLIENT_KEY//\\/\\\\}|" \
  | sed "s|TLS_CA|${MONGODB_CA//\\/\\\\}|" \
  | tee /dev/stderr
  set -x
)
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: dashboard-retweets
data:
  retweets.json: |
$(set +x; while IFS= read -r line; do echo "    ${line}" ; done < integration-test/dashboards/retweets.json; set -x)
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: dashboard-transactions
data:
  transactions.json: |
$(set +x; while IFS= read -r line; do echo "    ${line}" ; done < integration-test/dashboards/transactions.json; set -x)
EOF


GRAFANA_ARGS=(
    --set datasources.secretName=datasources
    --set admin.password=adminPassword
    --set config.grafanaIniConfigMap=grafana-ini
    --set config.useGrafanaIniFile=true
    --set dashboardsProvider.enabled=true
    --set dashboardsConfigMaps[0].configMapName=dashboard-retweets
    --set dashboardsConfigMaps[0].fileName=retweets.json
    --set dashboardsConfigMaps[1].configMapName=dashboard-transactions
    --set dashboardsConfigMaps[1].fileName=transactions.json
    # --set image.tag=7.1.5-debian-10-r9
)

if [ -n "${INTEGRATION_TEST_DEV_MODE:-}" ]; then
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



helm upgrade --install --wait grafana bitnami/grafana "${GRAFANA_ARGS[@]}"

        
if [ -n "${INTEGRATION_TEST_DEV_MODE:-}" ]; then
    kubectl rollout restart deploy/grafana
    kubectl rollout status deploy/grafana
    sleep 5
    echo 'Forwarding ports. Press Ctrl+C to exit and re-run this script to make changes'
    kubectl port-forward deploy/grafana 3000:3000
    exit 0
fi

fi
    

kubectl wait deploy/grafana --for=condition=available --timeout=300s
kubectl replace --force -f - <<'EOF'

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
        image: docker.io/library/centos:7 #docker.io/alpine/curl:latest
        command: [bash, -exuc]
        args:
        - |
            DATASOURCES=( $(seq 5) )
            for datasource in "${DATASOURCES[@]}"; do
                CMD=(
                    curl "http://grafana:3000/api/datasources/${datasource}/health"
                        -v
                        -u admin:adminPassword 
                )
                if ! "${CMD[@]}" --fail ; then
                    "${CMD[@]}"
                    exit 1
                fi
            done
            QUERY_DIR=/mnt/host/grafana-mongodb-community-plugin/integration-test/queries
            QUERIES=( weather/timeseries weather/timeseries-date weather/table tweets/timeseries )
            for query in "${QUERIES[@]}"; do
                QUERY_FILE="${QUERY_DIR}/${query}.json"
                CMD=(
                    curl 'http://grafana:3000/api/ds/query'
                      -v
                      -u admin:adminPassword
                      -H 'accept: application/json, text/plain, */*'
                      -H 'content-type: application/json'
                      --data "@${QUERY_FILE}"
                )

                if ! "${CMD[@]}" --fail ; then
                    "${CMD[@]}"
                    exit 1
                fi
            done
        volumeMounts:
        - name: datasets
          mountPath: /mnt/host/grafana-mongodb-community-plugin/integration-test/
      volumes:
      - name: datasets
        hostPath:
          path: /mnt/host/grafana-mongodb-community-plugin/integration-test/
         
EOF


(
    kubectl wait job/grafana-mongodb-community-plugin-it --for=condition=complete --timeout=-1s &
    ( kubectl wait job/grafana-mongodb-community-plugin-it --for=condition=failed --timeout=-1s && exit 1 ) &

    wait -n
) &
WAIT_PID=$!

(
    while ! kubectl get pods -ljob-name=grafana-mongodb-community-plugin-it | grep -E 'Running|Completed|Error' ; do
        sleep 1;
    done
    kubectl logs job/grafana-mongodb-community-plugin-it -f 
) &
LOGS_PID=$!

if ! wait "${WAIT_PID}"; then
    echo 'Tests failed'
    kill $(jobs -p -r)
    exit 1
fi

kill $(jobs -p -r) || true

echo "Tests passed"
