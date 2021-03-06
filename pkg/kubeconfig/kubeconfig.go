// Copyright Jetstack Ltd. See LICENSE for details.
package kubeconfig

import (
	"github.com/jetstack/vault-helper/pkg/cert"

	"github.com/sirupsen/logrus"
)

type Kubeconfig struct {
	configPath string
	certKey64  string
	certCA64   string
	cert64     string

	cert *cert.Cert
	Log  *logrus.Entry
}

func New(logger *logrus.Entry, c *cert.Cert) *Kubeconfig {
	u := &Kubeconfig{
		cert: c,
	}

	if logger != nil {
		u.Log = logger
	}

	return u
}

func (u *Kubeconfig) RunKube() error {
	if err := u.EncodeCerts(); err != nil {
		return err
	}

	yml, err := u.BuildYaml()
	if err != nil {
		return err
	}

	return u.StoreYaml(yml)
}

func (u *Kubeconfig) ConfigPath() string {
	return u.configPath
}
func (u *Kubeconfig) SetKubeConfigPath(path string) {
	u.configPath = path
}

func (u *Kubeconfig) SetCert(cert *cert.Cert) {
	u.cert = cert
}
func (u *Kubeconfig) Cert() (c *cert.Cert) {
	return u.cert
}

func (u *Kubeconfig) SetCertCA64(byt string) {
	u.certCA64 = byt
}
func (u *Kubeconfig) CertCA64() (byt string) {
	return u.certCA64
}

func (u *Kubeconfig) SetCertKey64(byt string) {
	u.certKey64 = byt
}
func (u *Kubeconfig) CertKey64() (byt string) {
	return u.certKey64
}

func (u *Kubeconfig) SetCert64(byt string) {
	u.cert64 = byt
}
func (u *Kubeconfig) Cert64() (byt string) {
	return u.cert64
}
