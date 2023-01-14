# Grafana Community MongoDB Plugin

This Open Source Grafana plugin allows for querying a MongoDB database or cluster.

# This plugin is still in early development, and experimental. Everything is subject to change. Use at your own risk. Help Wanted.

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
yarn backend
```

## Installation

See `integration-tests` directory for configuration examples.

### Production

Run

```bash
yarn plugin
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

### Development

Copy built repository to `<grafana plugins dir>/meln5674-mongodb-community`

## Integration Tests

Tools Needed:
* Docker
* KinD
* Helm
* Kubectl

```bash
yarn integration-test
```

## Live Development Environment

```bash
export KUBECONFIG=integration-test/kubeconfig
export INTEGRATION_TEST_NO_CLEANUP=1
export INTEGRATION_TEST_DEV_MODE=1
yarn integration-test
```

Grafana credentials: admin/adminPassword

MongoDD credentials: root/rootPassword

MongoDB Test datasets:

`test.weather { "metadata": { "sensorId": int, "type": string }, "timestamp": ISODate(...), "value": int }`

Cleanup:

```bash
export INTEGRATION_TEST_NO_CLEANUP=
export INTEGRATION_TEST_DEV_MODE=
yarn integration-test
```

## Limitations

* Only `aggregate` queries are supported, but you can emulate find() et al using various pipeline stages. Supporting find would likely require implementing a graphical query editor to prevent fragility.
* Grafana's data system requires that all values in a column be the same type. As such, queries from this plugin expect that a field will have the same type in all returned documents.
* Currently, you need to specify the types of each value field. This will hopefully be addressed in a later update to enable schema inference.
* Grafana only allows label values to be strings. For performance, this plugin considers, for example, integer 0 and string "0" to be the same label.
* Only anonymous and Username/Password authentication is supported.

## Help Wanted

Do you know about any of the following topics? If so, I'd love to hear from you!

* React.js - Making the query editor UI not an eyesore
* MongoDB - Providing representative sample data sets and queries to improve automated tests. Implementing other authentication types.
* Grafana - Sample dashboards using the sample datasets
* All three - Investigating implementing a graphical query builder.
* Github - Implementing automated releases

## Acknowledgements

Thank you to the following individuals/groups

* [James Osgood](https://github.com/JamesOsgood/) for writing the original plugin which inspired this one
* [Nikolay Stankov](https://github.com/nstankov-bg/) for PRs [1](https://github.com/meln5674/grafana-mongodb-community-plugin/pull/1)
