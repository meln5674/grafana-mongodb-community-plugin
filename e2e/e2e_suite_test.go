package e2e_test

import (
	"context"
	"fmt"
	"hash/adler32"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/meln5674/gingk8s"
	"github.com/meln5674/gosh"
	"github.com/onsi/biloba"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var b *biloba.Biloba
var gk8s *gingk8s.Gingk8s
var clusterHTTPClient *http.Client

var _ = BeforeSuite(func(ctx context.Context) {
	f, err := os.Create("../integration-test/datasets/download/tweets.zip")
	resp, err := http.Get("https://github.com/ozlerhakan/mongodb-json-files/blob/master/datasets/tweets.zip?raw=true")
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	_, err = io.Copy(f, resp.Body)
	Expect(err).ToNot(HaveOccurred())

	f, err = os.Create("../integration-test/datasets/download/transactions.json")
	resp, err = http.Get("https://github.com/fieldsets/mongodb-sample-datasets/blob/main/sample_analytics/transactions.json?raw=true")
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	_, err = io.Copy(f, resp.Body)
	Expect(err).ToNot(HaveOccurred())

	Expect(gosh.Command("unzip", "-o", "../integration-test/datasets/download/tweets.zip", "-d", "../integration-test/datasets/download/tweets/").WithStreams(gingk8s.GinkgoOutErr).Run()).To(Succeed())

	gk8s := gingk8s.ForSuite(GinkgoT())

	ingressNginxImageID := gk8s.ThirdPartyImage(ingressNginxImage)
	kubeIngressProxyImageID := gk8s.ThirdPartyImage(kubeIngressProxyImage)
	certManagerImageIDs := gk8s.ThirdPartyImages(certManagerImages...)

	clusterID := gk8s.Cluster(&cluster, ingressNginxImageID, kubeIngressProxyImageID, certManagerImageIDs)

	gk8s.ClusterAction(clusterID, "Watch Pods", &watchPods)

	certManagerID := gk8s.Release(clusterID, &certManager, certManagerImageIDs)

	mongodbInitID := gk8s.Manifests(clusterID, &mongodbInit, certManagerID)

	mongodbCertSecretID := gk8s.Manifests(clusterID, &mongodbCertSecret, mongodbInitID)

	// gk8s.ClusterAction(clusterID, "MongoDB Logs", &gingk8s.KubectlLogger{Kind: "sts", Name: "mongodb", RetryPeriod: 5 * time.Second})
	// gk8s.ClusterAction(clusterID, "MongoDB (No Auth) Logs", &gingk8s.KubectlLogger{Kind: "sts", Name: "mongodb-no-auth", RetryPeriod: 5 * time.Second})
	// gk8s.ClusterAction(clusterID, "MongoDB (TLS) Logs", &gingk8s.KubectlLogger{Kind: "sts", Name: "mongodb-tls", RetryPeriod: 5 * time.Second})
	// gk8s.ClusterAction(clusterID, "MongoDB (mTLS) Logs", &gingk8s.KubectlLogger{Kind: "sts", Name: "mongodb-mtls", RetryPeriod: 5 * time.Second})

	gk8s.Release(clusterID, &mongodb, mongodbInitID)

	gk8s.Release(clusterID, &mongodbNoAuth)

	gk8s.Release(clusterID, &mongodbTLS, mongodbInitID, mongodbCertSecretID)

	gk8s.Release(clusterID, &mongodbMTLS, mongodbCertSecretID, mongodbCertSecretID)

	grafanaResourcesID := gk8s.Manifests(clusterID, &grafanaResources, mongodbCertSecretID)

	ingressNginxID := gk8s.Release(clusterID, &ingressNginx, ingressNginxImageID)

	grafanaDeps := []gingk8s.ResourceDependency{mongodbInitID, grafanaResourcesID, ingressNginxID}
	if !devMode {
		pluginRepoID := gk8s.Release(clusterID, &pluginRepo)
		grafanaDeps = append(grafanaDeps, pluginRepoID)
	}

	gk8s.Release(clusterID, &kubeIngressProxy, ingressNginxID, kubeIngressProxyImageID)

	gk8s.Release(clusterID, &grafana, grafanaDeps...)

	gk8s.Options(gingk8s.SuiteOpts{
		NoSuiteCleanup: os.Getenv("INTEGRATION_TEST_NO_CLEANUP") != "",
		NoSpecCleanup:  os.Getenv("INTEGRATION_TEST_NO_CLEANUP") != "",
	})
	gk8s.Setup(ctx)

	bopts := []chromedp.ExecAllocatorOption{
		chromedp.ProxyServer("http://localhost:8080"),
		chromedp.WindowSize(1920, 1080),
	}

	if os.Getenv("IT_IN_CONTAINER") != "" {
		bopts = append(bopts, chromedp.NoSandbox)
		GinkgoWriter.Printf("!!! WARNING: Sandbox disabled due to containerized environment detected from IT_IN_CONTAINER. This is insecure if this not actually a container!\n")
	}

	clusterHTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) {
				return url.Parse("http://localhost:8080")
			},
		},
	}

	Eventually(func(g gomega.Gomega) int {
		resp, err := clusterHTTPClient.Get("http://grafana.grafana-mongodb-it.cluster/login")
		g.Expect(err).ToNot(HaveOccurred())
		return resp.StatusCode
	}, "15s").Should(Equal(http.StatusOK))

	biloba.SpinUpChrome(GinkgoT(), bopts...)
	b = biloba.ConnectToChrome(GinkgoT())

	b.Navigate("http://grafana.grafana-mongodb-it.cluster")
	Eventually(b.Location).Should(Equal("http://grafana.grafana-mongodb-it.cluster/login"))
	Eventually(`input[name="user"]`).Should(b.Exist())
	// b.Click(`input[name="user"]`)
	// b.SetValue(`input[name="user"]`, "admin")
	// b.Click(`input[name="password"]`)
	// b.SetValue(`input[name="password"]`, "adminPassword")
	Expect(chromedp.Run(b.Context, chromedp.SendKeys(`input[name="user"]`, "admin"))).To(Succeed())
	Expect(chromedp.Run(b.Context, chromedp.SendKeys(`input[name="password"]`, "adminPassword"))).To(Succeed())
	b.Click(`button[aria-label="Login button"]`)
	Eventually(b.Location, "15s").Should(Equal("http://grafana.grafana-mongodb-it.cluster/?orgId=1"))
})

var (
	devMode = os.Getenv("INTEGRATION_TEST_DEV_MODE") != ""
)

var (
	cluster = gingk8s.KindCluster{
		Name:                   "grafana-mongodb",
		KindCommand:            gingk8s.DefaultKind,
		TempDir:                "../integration-test",
		ConfigFilePath:         "../integration-test/kind.config",
		ConfigFileTemplatePath: "../integration-test/kind.config.template",
	}

	watchPods = gingk8s.KubectlWatcher{
		Kind:  "pods",
		Flags: []string{"--all-namespaces"},
	}

	certManagerImages = []*gingk8s.ThirdPartyImage{
		&gingk8s.ThirdPartyImage{Name: "quay.io/jetstack/cert-manager-cainjector:v1.11.1"},
		&gingk8s.ThirdPartyImage{Name: "quay.io/jetstack/cert-manager-controller:v1.11.1"},
		&gingk8s.ThirdPartyImage{Name: "quay.io/jetstack/cert-manager-webhook:v1.11.1"},
	}
	certManager = gingk8s.HelmRelease{
		Name: "cert-manager",
		Chart: &gingk8s.HelmChart{
			RemoteChartInfo: gingk8s.RemoteChartInfo{
				Repo: &gingk8s.HelmRepo{
					Name: "jetstack",
					URL:  "https://charts.jetstack.io",
				},
				Name:    "cert-manager",
				Version: "v1.11.1",
			},
		},
		Set: gingk8s.Object{
			"installCRDs":        true,
			"prometheus.enabled": false,
		},
	}

	ingressNginxImage = &gingk8s.ThirdPartyImage{Name: "registry.k8s.io/ingress-nginx/controller:v1.7.0"}
	ingressNginx      = gingk8s.HelmRelease{
		Name: "ingress-nginx",
		Chart: &gingk8s.HelmChart{
			RemoteChartInfo: gingk8s.RemoteChartInfo{
				Repo: &gingk8s.HelmRepo{
					Name: "ingress-nginx",
					URL:  "https://kubernetes.github.io/ingress-nginx",
				},
				Name:    "ingress-nginx",
				Version: "4.6.0",
			},
		},
		Values: []gingk8s.NestedObject{
			{
				"controller": gingk8s.NestedObject{
					"service": gingk8s.NestedObject{
						"type": "ClusterIP",
					},
				},
			},
		},
	}

	kubeIngressProxyImage = &gingk8s.ThirdPartyImage{Name: "ghcr.io/meln5674/kube-ingress-proxy:v0.3.0-rc1"}
	kubeIngressProxy      = gingk8s.HelmRelease{
		Name: "kube-ingress-proxy",
		Chart: &gingk8s.HelmChart{
			RemoteChartInfo: gingk8s.RemoteChartInfo{
				Name: "kube-ingress-proxy",
				Repo: &gingk8s.HelmRepo{
					Name: "kube-ingress-proxy",
					URL:  "https://meln5674.github.io/kube-ingress-proxy",
				},
				Version: "v0.3.0-rc1",
			},
		},
		Set: gingk8s.Object{
			"controllerAddresses[0].className": "nginx",
			"controllerAddresses[0].address":   "ingress-nginx-controller.default.svc.cluster.local",
			"hostPort.enabled":                 "true",
		},
		NoWait: true,
	}

	mongodbInit = gingk8s.KubernetesManifests{
		Name: "MongoDB Certificates and Datasets",
		ResourceObjects: []interface{}{
			mongodbDatasets(),
		},
		ResourcePaths: []string{"../integration-test/mongodb-certs.yaml"},
		Wait: []gingk8s.WaitFor{
			{
				Resource: "certificate/mongodb-tls-ca",
				For:      map[string]string{"condition": "ready"},
			},
			{
				Resource: "certificate/mongodb-mtls-client",
				For:      map[string]string{"condition": "ready"},
			},
		},
	}

	mongodbCertSecretName = "mongodb-tls-certs"

	mongodbCertSecret = gingk8s.KubernetesManifests{
		Name: "MongoDB Certificate Secret",
		ResourceObjects: []interface{}{
			gingk8s.Secret(
				mongodbCertSecretName,
				"",
				"Opaque",
				nil,
				gingk8s.Object{
					"mongodb-ca-cert": func(g gingk8s.Gingk8s, ctx context.Context, cluster gingk8s.Cluster) string {
						return g.KubectlReturnSecretValue(ctx, cluster, "mongodb-tls-ca", "tls.crt")
					},

					"mongodb-ca-key": func(g gingk8s.Gingk8s, ctx context.Context, cluster gingk8s.Cluster) string {
						return g.KubectlReturnSecretValue(ctx, cluster, "mongodb-tls-ca", "tls.key")
					},
				},
			),
		},
	}

	bitnamiRepo = gingk8s.HelmRepo{
		Name: "bitnami",
		URL:  "https://charts.bitnami.com/bitnami",
	}

	bitnamiLegacyRepo = gingk8s.HelmRepo{
		Name: "bitnami-legacy",
		URL:  "https://raw.githubusercontent.com/bitnami/charts/archive-full-index/bitnami",
	}

	mongodbSet = gingk8s.Object{
		"auth.rootPassword":              "rootPassword",
		"initdbScriptsConfigMap":         "mongodb-init",
		"useStatefulSet":                 true,
		"extraVolumes[0].name":           "sample-data",
		"extraVolumes[0].hostPath.path":  "/mnt/host/grafana-mongodb-community-plugin/integration-test/datasets/download",
		"extraVolumeMounts[0].name":      "sample-data",
		"extraVolumeMounts[0].mountPath": "/mnt/host/grafana-mongodb-community-plugin/integration-test/datasets/download",
		"image.debug":                    true,
	}

	mongodbChart = gingk8s.HelmChart{
		/*
			RemoteChartInfo: gingk8s.RemoteChartInfo{
				Name:    "mongodb",
				Repo:    &bitnamiRepo,
				Version: "13.18.1",
			}
		*/
		LocalChartInfo: gingk8s.LocalChartInfo{
			Path: "../integration-test/bitnami-charts/bitnami/mongodb/",
			// DependencyUpdate: true,
		},
	}

	mongodb = gingk8s.HelmRelease{
		Name:  "mongodb",
		Chart: &mongodbChart,
		Set:   mongodbSet,
	}

	mongodbNoAuth = gingk8s.HelmRelease{
		Name:  "mongodb-no-auth",
		Chart: &mongodbChart,
		Set:   mongodbSet.With("auth.enabled", false),
	}

	mongodbMTLSSet = mongodbSet.MergedFrom(gingk8s.Object{
		"tls.enabled":        true,
		"tls.existingSecret": mongodbCertSecretName,
	})

	mongodbMTLS = gingk8s.HelmRelease{
		Name:  "mongodb-mtls",
		Chart: &mongodbChart,
		Set:   mongodbMTLSSet,
	}

	mongodbTLS = gingk8s.HelmRelease{
		Name:  "mongodb-tls",
		Chart: &mongodbChart,
		Set:   mongodbMTLSSet.With("tls.mTLS.enabled", false),
	}

	pluginRepo = gingk8s.HelmRelease{
		Name: "plugin-repo",
		Chart: &gingk8s.HelmChart{
			RemoteChartInfo: gingk8s.RemoteChartInfo{
				Name: "nginx",
				Repo: &bitnamiRepo,
				// TODO: Pin version
			},
		},
		Set: gingk8s.Object{
			"fullnameOverride":               "plugin-repo",
			"extraVolumes[0].name":           "plugin",
			"extraVolumes[0].hostPath.path":  "/mnt/host/grafana-mongodb-community-plugin/",
			"extraVolumeMounts[0].name":      "plugin",
			"extraVolumeMounts[0].mountPath": "/opt/bitnami/nginx/html/grafana-mongodb-community-plugin/",
			"service.type":                   "ClusterIP",
		},
	}

	grafanaResources = gingk8s.KubernetesManifests{
		ResourceObjects: []interface{}{
			gingk8s.Secret("datasources", "", "Opaque",
				gingk8s.Object{
					"datasources.yaml": datasourcesYAML,
				},
				nil,
			),
			gingk8s.ConfigMap("dashboard-retweets", "", nil,
				gingk8s.Object{
					"retweets.json": func() ([]byte, error) {
						return os.ReadFile("../integration-test/dashboards/retweets.json")
					},
				},
			),
			gingk8s.ConfigMap("dashboard-transactions", "", nil,
				gingk8s.Object{
					"transactions.json": func() ([]byte, error) {
						return os.ReadFile("../integration-test/dashboards/transactions.json")
					},
				},
			),
			gingk8s.ConfigMap("grafana-ini", "", nil,
				gingk8s.Object{
					"grafana.ini": func() ([]byte, error) {
						return os.ReadFile("../integration-test/grafana.ini")
					},
				},
			),
		},
	}

	grafanaBaseSet = gingk8s.Object{
		"datasources.secretName":                "datasources",
		"admin.password":                        "adminPassword",
		"config.grafanaIniConfigMap":            "grafana-ini",
		"config.useGrafanaIniFile":              true,
		"dashboardsProvider.enabled":            true,
		"dashboardsConfigMaps[0].configMapName": "dashboard-retweets",
		"dashboardsConfigMaps[0].fileName":      "retweets.json",
		"dashboardsConfigMaps[1].configMapName": "dashboard-transactions",
		"dashboardsConfigMaps[1].fileName":      "transactions.json",
		"podLabels.plugin-sum": func() string {
			pluginBytes, err := os.ReadFile("../meln5674-mongodb-community.zip")
			Expect(err).ToNot(HaveOccurred())
			return fmt.Sprintf("%d", adler32.Checksum(pluginBytes))
		},
		"extraEnvVars[0].name":     "GF_DEFAULT_APP_MODE",
		"extraEnvVars[0].value":    "development",
		"updateStrategy.type":      "Recreate",
		"ingress.enabled":          true,
		"ingress.hostname":         "grafana.grafana-mongodb-it.cluster",
		"ingress.ingressClassName": "nginx",
	}

	grafanaDevModeSetExtra = gingk8s.Object{
		"grafana.extraVolumes[0].name":           "plugin",
		"grafana.extraVolumes[0].hostPath.path":  "/mnt/host/grafana-mongodb-community-plugin/",
		"grafana.extraVolumeMounts[0].name":      "plugin",
		"grafana.extraVolumeMounts[0].mountPath": "/opt/bitnami/grafana/data/plugins/meln5674-mongodb-community",
	}

	grafanaNonDevModeSetExtra = gingk8s.Object{
		"plugins": "meln5674-mongodb-community=http://plugin-repo/grafana-mongodb-community-plugin/meln5674-mongodb-community.zip",
	}

	grafana = gingk8s.HelmRelease{
		Name: "grafana",
		Chart: &gingk8s.HelmChart{
			RemoteChartInfo: gingk8s.RemoteChartInfo{
				Name: "grafana",
				Repo: &bitnamiRepo,
				// TODO: Pin Version
			},
		},
		Set: grafanaSet(devMode),
	}

	grafana7 = gingk8s.HelmRelease{
		Name: "grafana-7",
		Chart: &gingk8s.HelmChart{
			RemoteChartInfo: gingk8s.RemoteChartInfo{
				Name:    "grafana",
				Repo:    &bitnamiLegacyRepo,
				Version: "5.2.19",
			},
		},
		Set: grafanaSet(devMode),
	}

	grafana8 = gingk8s.HelmRelease{
		Name: "grafana-8",
		Chart: &gingk8s.HelmChart{
			RemoteChartInfo: gingk8s.RemoteChartInfo{
				Name:    "grafana",
				Repo:    &bitnamiRepo,
				Version: "7.9.11",
			},
		},
		Set: grafanaSet(devMode),
	}

	grafanaPortForward = gingk8s.KubectlPortForwarder{
		Kind:  "service",
		Name:  "grafana",
		Ports: []string{"3000:3000"},
	}
)

func grafanaSet(devMode bool) gingk8s.Object {
	if devMode {
		return grafanaBaseSet.MergedFrom(grafanaDevModeSetExtra)
	} else {
		return grafanaBaseSet.MergedFrom(grafanaNonDevModeSetExtra)
	}
}

func datasourcesYAML(g gingk8s.Gingk8s, ctx context.Context, cluster gingk8s.Cluster) ([]byte, error) {
	datasourcesBytes, err := os.ReadFile("../integration-test/datasources.yaml")
	Expect(err).ToNot(HaveOccurred())
	datasources := map[string]interface{}{}
	err = yaml.Unmarshal(datasourcesBytes, &datasources)
	Expect(err).ToNot(HaveOccurred())
	ca := g.KubectlReturnSecretValue(ctx, cluster, "mongodb-tls-certs", "mongodb-ca-cert")
	cert := g.KubectlReturnSecretValue(ctx, cluster, "mongodb-mtls-client", "tls.crt")
	key := g.KubectlReturnSecretValue(ctx, cluster, "mongodb-mtls-client", "tls.key")

	mTLSSource := datasources["datasources"].([]interface{})[2].(map[string]interface{})
	mTLSSourceData := mTLSSource["jsonData"].(map[string]interface{})
	mTLSSourceSecureData := mTLSSource["secureJsonData"].(map[string]interface{})
	tlsSource := datasources["datasources"].([]interface{})[3].(map[string]interface{})
	tlsSourceData := tlsSource["jsonData"].(map[string]interface{})

	tlsSourceData["tlsCa"] = ca
	mTLSSourceData["tlsCa"] = ca
	mTLSSourceData["tlsCertificate"] = cert
	mTLSSourceSecureData["tlsCertificateKey"] = key
	return yaml.Marshal(&datasources)
}

func mongodbDatasets() gingk8s.NestedObject {
	binaryData := gingk8s.Object{}
	for _, dataset := range []string{"weather.js", "tweets.sh", "transactions.sh", "conversion_check.js"} {
		binaryData[dataset] = func(dataset string) func() ([]byte, error) {
			return func() ([]byte, error) {
				return os.ReadFile(filepath.Join("../integration-test/datasets", dataset))
			}
		}(dataset)
	}
	return gingk8s.ConfigMap("mongodb-init", "", nil, binaryData)
}
