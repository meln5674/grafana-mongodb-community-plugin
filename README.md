# Grafana Community MongoDB Plugin

This Open Source Grafana plugin allows for querying a MongoDB database or cluster.

# This plugin is still in early development, and experimental. Everything is subject to change. Use at your own risk. Help Wanted.


## Installation


This plugin is not currently signed. To install it, you have two options: Sign it yourself, or whitelist it as an unsigned plugin.

### Signed

This is recommended for production environments.

To sign the plugin, build the plugin from source as described below, then execute

```bash
export GRAFANA_API_KEY=<See https://grafana.com/docs/grafana/latest/developers/plugins/sign-a-plugin/#generate-an-api-key>
yarn plugin
yarn sign -- --rootUrls=<your grafana URL>
```

Then copy the produced `meln5674-mongodb-community.zip` file to your artifact repository (e.g. Nexus) or web server (.e.g Nginx) of choice, and note the URL to download the zip.

### Unsigned

To install the plugin as unsigned, choose a version from the [Releases Page](https://github.com/meln5674/grafana-mongodb-community-plugin/releases), and either download the ZIP file, or copy its URL.

### Installation methods

To view examples of installing the plugin see [this directory](./examples). All examples use the `${ZIP_URL}` variable to refer to either a URL from the releases page, or the URL your signed plugin is accessible from.

### Development

#### Building

Tools Needed:
* Node.js 14+
* Golang 1.17+
* Yarn
* Mage

If you have Docker, installed you can use `./build-env.sh` to build and run a shell in a container with all necessary tools (or `build-env.Dockerfile` to build it yourself manually). You can also execute `./build-env.sh <command>` to execute a single command, batch-style, in this container.

To build, run

```bash
yarn install
yarn build
yarn backend
```

then to install into a development environment, copy built repository to `<grafana plugins dir>/meln5674-mongodb-community`

#### Integration Tests

Tools Needed:
* Docker
* KinD
* Helm
* Kubectl

```bash
yarn integration-test
```

#### Live Development Environment

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
