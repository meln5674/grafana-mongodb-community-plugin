# Grafna Community MongoDB Plugin

This Open Source Grafana plugin allows for querying with a MongoDB database or cluster.

## Building

Tools Needed:
* Node.js 14+
* Golang 1.17+
* Yarn
* Mage

```bash
yarn install
yarn dev
mage -v
```

## Installation

Copy built repository to `<grafana plugins dir>/meln5674-mongodb-community`

See `integration-tests` directory for configuration examples.

## Integration Tests

Tools Needed:
* Docker
* KinD
* Helm
* Kubectl

```bash
integration-test/run.sh
```

## Live Development Environment

```bash
export KUBECONFIG=integration-test/kubeconfig
export INTEGRATION_TEST_NO_CLEANUP=1
export INTEGRATION_TEST_DEV_MODE=1
integration-test/run.sh
kubectl port-forward deploy/grafana 3000:3000
```

Grafana credentials: admin/admin
MongoDD credentials: root/root
MongoDB Test dataset: test.weather { "metadata": { "sensorId": int, "type": string }, "timestamp": ISODate(...), "value": float }

Cleanup:

```bash
export INTEGRATION_TEST_NO_CLEANUP=
export INTEGRATION_TEST_DEV_MODE=
integration-test/run.sh
```
