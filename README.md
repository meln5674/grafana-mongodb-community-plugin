# Grafana Community MongoDB Plugin

This Open Source Grafana plugin allows for querying a MongoDB database or cluster.

## Building

Tools Needed:
* Node.js 14+
* Golang 1.17+
* Yarn
* Mage

If you have Docker installed you can use `./build-env.sh` to build and run a shell in a container with all necessary tools (or `build-env.Dockerfile` to build it yourself manually).

```bash
yarn install
yarn build
mage -v
```

## Installation

See `integration-tests` directory for configuration examples.

### Production

Run

```bash
yarn make-plugin
```

Then copy the produced `meln5674-mongodb-community.zip` file to your artifact repository (e.g. Nexus) or web server (.e.g Nginx) of choice.

#### Bare Metal

Run

```bash
grafana-cli --pluginUrl <your repository url>/meln5674-mongodb-community.zip meln5674-mongodb-community
```

on your grafana host.

#### Docker

Set the environment variable

```
GF_INSTALL_PLUGINS=meln5674-mongodb-community=<your repository url>/meln5674-mongodb-community.zip
```

#### Kubernetes

Consult your grafana distribution documentation (e.g. https://github.com/bitnami/charts/tree/master/bitnami/grafana) for how to specify plugins to install

#### Development

Copy built repository to `<grafana plugins dir>/meln5674-mongodb-community`

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
