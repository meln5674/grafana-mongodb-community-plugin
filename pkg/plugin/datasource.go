package plugin

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
)

type jsonData struct {
	URL            string `json:"url"`
	TLS            bool   `json:"tls"`
	TLSCertificate string `json:"tlsCertificate"`
	TLSCA          string `json:"tlsCa"`
	TLSInsecure    bool   `json:"tlsInsecure"`
	TLSServerName  string `json:"tlsServerName"`
}

type secureJsonData struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	TLSCertificateKey string `json:"tlsCertificateKey"`
}

type datasource struct {
	jsonData
	secureJsonData
}

func (d *datasource) applyAuth(uri *url.URL) error {
	if d.Username == "" {
		return nil
	}
	/*if d.Password != "" {
		mongoURL.User = url.UserPassword(d.Username, d.Password)
	} else {
		mongoURL.User = url.User(d.Username)
	}*/

	uri.User = url.UserPassword(d.Username, d.Password)

	return nil
}

func (d *datasource) getTLS() (*tls.Config, error) {
	if !d.TLS {
		return nil, nil
	}
	tlsConfig := &tls.Config{}
	if d.TLSCA != "" {
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM([]byte(d.TLSCA)) {
			return nil, fmt.Errorf("failed to add tlsCA")
		}
	}
	if d.TLSInsecure {
		tlsConfig.InsecureSkipVerify = true
	}
	if (d.TLSCertificate != "") != (d.TLSCertificateKey != "") {
		return nil, fmt.Errorf("Must provide both tlsCertificate and tlsCertificateKey, or neither")
	}
	if d.TLSCertificate != "" && d.TLSCertificateKey != "" {
		clientCert, err := tls.X509KeyPair([]byte(d.TLSCertificate), []byte(d.TLSCertificateKey))
		if err != nil {
			return nil, errors.Wrap(err, "Failed to parse TLS Certificate-Key Pair")
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}
	if d.TLSServerName != "" {
		tlsConfig.ServerName = d.TLSServerName
	}
	return tlsConfig, nil
}
