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

### Bare Metal

On your grafana server, run

```bash
grafana-cli --pluginUrl ${ZIP_URL} meln5674-mongodb-community
```

then, if using the plugin unsigned, add the following to your `grafana.ini` file:

```ini
[plugins]
allow_loading_unsigned_plugins=meln5674-mongodb-community
```

#### Docker

Set the environment variable

```bash
GF_INSTALL_PLUGINS=${ZIP_URL}
```

as well as the following, if using the plugin unsigned
```bash
GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=meln5674-mongodb-community
```

e.g.

```bash
PLUGIN_VERSION=<See https://github.com/meln5674/grafana-mongodb-community-plugin/releases>
docker run \
    -d \
    -p 3000:3000 \
    -e GF_INSTALL_PLUGINS=meln5674-mongodb-community=https://github.com/meln5674/grafana-mongodb-community-plugin/releases/download/${PLUGIN_VERSION}/meln5674-mongodb-community.zip \
    -e GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=meln5674-mongodb-community \
    bitnami/grafana:latest
```

#### Kubernetes

Consult your grafana distribution documentation (e.g. https://github.com/bitnami/charts/tree/master/bitnami/grafana) for how to specify plugins to install.

For a simple deployment, set the following

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
spec:
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      restartPolicy: Always
      containers:
      - name: grafana
        image: bitnami/grafana:latest
        env:
        - name: PLUGIN_VERSION
          value: <See https://github.com/meln5674/grafana-mongodb-community-plugin/releases>
        - name: GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS
          value: meln5674-mongodb-community
        - name: GF_INSTALL_PLUGINS
          value: meln5674-mongodb-community=https://github.com/meln5674/grafana-mongodb-community-plugin/releases/download/$(PLUGIN_VERSION)/meln5674-mongodb-community.zip
```

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
