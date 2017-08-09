package cert

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/jetstack-experimental/vault-helper/pkg/instanceToken"
	"github.com/jetstack-experimental/vault-helper/pkg/kubernetes"
	"github.com/jetstack-experimental/vault-helper/pkg/testing/vault_dev"
)

var vaultDev *vault_dev.VaultDev

var tempDirs []string

func TestMain(m *testing.M) {
	vaultDev = initVaultDev()

	// this runs all tests
	returnCode := m.Run()

	// shutdown vault
	vaultDev.Stop()

	// clean up tempdirs
	for _, dir := range tempDirs {
		os.RemoveAll(dir)
	}

	// return exit code according to the test runs
	os.Exit(returnCode)
}

// Test permissons of created files
func TestCert_File_Perms(t *testing.T) {

	k := initKubernetes(t, vaultDev)
	c, i := initCert(t, vaultDev)

	token := k.InitTokens()["master"]
	if err := i.WriteTokenFile(i.InitTokenFilePath(), token); err != nil {
		t.Fatalf("error setting token for test: %s", err)
	}

	if err := c.RunCert(); err != nil {
		t.Fatalf("error runinning cert: %s", err)
	}

	dir := filepath.Dir(c.Destination())
	if fi, err := os.Stat(dir); err != nil {
		t.Fatalf("error finding stats of '%s': %s", dir, err)
	} else if !fi.IsDir() {
		t.Fatalf("destination should be directory %s. It is not", dir)
	} else if perm := fi.Mode(); perm.String() != "drwxr-xr-x" {
		t.Fatalf("destination has incorrect file permissons. exp=drwxr-xr-x got=%s", perm)
	}

	keyPem := filepath.Clean(c.Destination() + "-key.pem")
	dotPem := filepath.Clean(c.Destination() + ".pem")
	caPem := filepath.Clean(c.Destination() + "-ca.pem")
	checkFilePerm(t, keyPem, os.FileMode(0600))
	checkFilePerm(t, dotPem, os.FileMode(0644))
	checkFilePerm(t, caPem, os.FileMode(0644))
}

// Check permissions of a file
func checkFilePerm(t *testing.T, path string, mode os.FileMode) {
	if fi, err := os.Stat(path); err != nil {
		t.Fatalf("error finding stats of '%s': %s", path, err)
	} else if fi.IsDir() {
		t.Fatalf("file should not be directory %s", path)
	} else if perm := fi.Mode(); perm != mode {
		t.Fatalf("destination has incorrect file permissons. exp=%s got=%s", mode, perm)
	}

}

// Verify CAs exist
func TestCert_Verify_CA(t *testing.T) {

	k := initKubernetes(t, vaultDev)
	c, i := initCert(t, vaultDev)
	token := k.InitTokens()["master"]
	if err := i.WriteTokenFile(i.InitTokenFilePath(), token); err != nil {
		t.Fatalf("failed to set token for test: %s", err)
	}

	if err := c.RunCert(); err != nil {
		t.Fatalf("error runinning cert: %s", err)
	}

	dotPem := filepath.Clean(c.Destination() + ".pem")
	dat, err := ioutil.ReadFile(dotPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", dotPem, err)
	}
	if dat == nil {
		t.Fatalf("no certificate at file '%s'. expected certificate", dotPem)
	}

	caPem := filepath.Clean(c.Destination() + "-ca.pem")
	dat, err = ioutil.ReadFile(dotPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", caPem, err)
	}
	if dat == nil {
		t.Fatalf("no certificate at file '%s'. expected certificate", dotPem)
	}
}

// Test config file path
func TestCert_ConfigPath(t *testing.T) {
	k := initKubernetes(t, vaultDev)

	dir, err := ioutil.TempDir("", "test-cluster-dir")
	if err != nil {
		t.Fatal(err)
	}

	c, i := initCert(t, vaultDev)
	i.SetVaultConfigPath(dir)
	c.SetVaultConfigPath(dir)
	token := k.InitTokens()["master"]
	if err := i.WriteTokenFile(i.InitTokenFilePath(), token); err != nil {
		t.Fatalf("error setting token for test: %s", err)
	}

	dotPem := filepath.Clean(c.Destination() + ".pem")
	if _, err := os.Stat(dotPem); !os.IsNotExist(err) {
		t.Fatalf("expexted error 'File doesn't exist on file '.pem''. got: %s", err)
	}

	if err := c.RunCert(); err != nil {
		t.Fatalf("error runinning cert: %s", err)
	}

	caPem := filepath.Clean(c.Destination() + "-ca.pem")
	if _, err := os.Stat(caPem); err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", caPem, err)
	}

	dat, err := ioutil.ReadFile(dotPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", dotPem, err)
	}
	if dat == nil {
		t.Fatalf("no certificate at file '%s'. expected certificate", dotPem)
	}

	caPem = filepath.Clean(c.Destination() + "-ca.pem")
	dat, err = ioutil.ReadFile(dotPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", caPem, err)
	}
	if dat == nil {
		t.Fatalf("no certificate at file '%s'. expected certificate", dotPem)
	}
}

// Test if already existing valid certificate and key, they are kept
func TestCert_Exist_NoChange(t *testing.T) {
	k := initKubernetes(t, vaultDev)

	dir, err := ioutil.TempDir("", "test-cluster-dir")
	if err != nil {
		t.Fatal(err)
	}

	c, i := initCert(t, vaultDev)
	i.SetVaultConfigPath(dir)
	c.SetVaultConfigPath(dir)
	token := k.InitTokens()["master"]
	if err := i.WriteTokenFile(i.InitTokenFilePath(), token); err != nil {
		t.Fatalf("error setting token for test: %s", err)
	}

	if err := c.RunCert(); err != nil {
		t.Fatalf("error running  cert: %s", err)
	}

	dotPem := filepath.Clean(c.Destination() + ".pem")
	datDotPem, err := ioutil.ReadFile(dotPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", dotPem, err)
	}
	if datDotPem == nil {
		t.Fatalf("no certificate at file '%s'. expected certificate", dotPem)
	}

	caPem := filepath.Clean(c.Destination() + "-ca.pem")
	datCAPem, err := ioutil.ReadFile(caPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", caPem, err)
	}
	if datCAPem == nil {
		t.Fatalf("no certificate at file '%s'. expected certificate", dotPem)
	}

	keyPem := filepath.Clean(c.Destination() + "-key.pem")
	datKeyPem, err := ioutil.ReadFile(keyPem)
	if err != nil {
		t.Fatalf("error reading from key file path: '%s': %s", keyPem, err)
	}
	if datKeyPem == nil {
		t.Fatalf("no key at file '%s'. expected key", keyPem)
	}

	c.Log.Infof("-- Second run call --")
	if err := c.RunCert(); err != nil {
		t.Fatalf("error running  cert: %s", err)
	}

	datDotPemAfter, err := ioutil.ReadFile(dotPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", dotPem, err)
	}

	if string(datDotPem) != string(datDotPemAfter) {
		t.Fatalf("certificate has been changed after cert call even though it exists. it shouldn't. %s", dotPem)
	}

	datCAPemAfter, err := ioutil.ReadFile(caPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", caPem, err)
	}
	if string(datCAPem) != string(datCAPemAfter) {
		t.Fatalf("certificate has been changed after cert call even though it exists. it shouldn't. %s", caPem)
	}

	datKeyPemAfter, err := ioutil.ReadFile(keyPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", keyPem, err)
	}
	if string(datKeyPem) != string(datKeyPemAfter) {
		t.Fatalf("key has been changed after cert call even though it exists. it shouldn't. %s", keyPem)
	}
}

func TestCert_Busy_Vault(t *testing.T) {
	k := initKubernetes(t, vaultDev)

	dir, err := ioutil.TempDir("", "test-cluster-dir")
	if err != nil {
		t.Fatal(err)
	}

	c, i := initCert(t, vaultDev)
	i.SetVaultConfigPath(dir)
	c.SetVaultConfigPath(dir)
	token := k.InitTokens()["master"]
	if err := i.WriteTokenFile(i.InitTokenFilePath(), token); err != nil {
		t.Fatalf("error setting token for test: %s", err)
	}

	if err := c.RunCert(); err != nil {
		t.Fatalf("error running  cert: %s", err)
	}

	dotPem := filepath.Clean(c.Destination() + ".pem")
	datDotPem, err := ioutil.ReadFile(dotPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", dotPem, err)
	}
	if datDotPem == nil {
		t.Fatalf("no certificate at file '%s'. expected certificate", dotPem)
	}

	caPem := filepath.Clean(c.Destination() + "-ca.pem")
	datCAPem, err := ioutil.ReadFile(caPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", caPem, err)
	}
	if datCAPem == nil {
		t.Fatalf("no certificate at file '%s'. expected certificate", dotPem)
	}

	keyPem := filepath.Clean(c.Destination() + "-key.pem")
	datKeyPem, err := ioutil.ReadFile(keyPem)
	if err != nil {
		t.Fatalf("error reading from key file path: '%s': %s", keyPem, err)
	}
	if datKeyPem == nil {
		t.Fatalf("no key at file '%s'. expected key", keyPem)
	}

	c.Log.Infof("-- Second run call --")
	c.vaultClient.SetToken("foo-bar")
	if err := c.RunCert(); err == nil {
		t.Fatalf("expected 400 error, premisson denied")
	}

	datDotPemAfter, err := ioutil.ReadFile(dotPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", dotPem, err)
	}

	if string(datDotPem) != string(datDotPemAfter) {
		t.Fatalf("certificate has been changed after cert call even though it exists. it shouldn't. %s", dotPem)
	}

	datCAPemAfter, err := ioutil.ReadFile(caPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", caPem, err)
	}
	if string(datCAPem) != string(datCAPemAfter) {
		t.Fatalf("certificate has been changed after cert call even though it exists. it shouldn't. %s", caPem)
	}

	datKeyPemAfter, err := ioutil.ReadFile(keyPem)
	if err != nil {
		t.Fatalf("error reading from certificate file path: '%s': %s", keyPem, err)
	}
	if string(datKeyPem) != string(datKeyPemAfter) {
		t.Fatalf("key has been changed after cert call even though it exists. it shouldn't. %s", keyPem)
	}

}

// Init Cert for tesing
func initCert(t *testing.T, vaultDev *vault_dev.VaultDev) (c *Cert, i *instanceToken.InstanceToken) {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	log := logrus.NewEntry(logger)

	c = New(vaultDev.Client(), log)
	c.SetRole("test-cluster/pki/k8s/sign/kube-apiserver")
	c.SetCommonName("k8s")
	c.SetBitSize(2048)

	if usr, err := user.Current(); err != nil {
		t.Fatalf("error getting info on current user: %s", err)
	} else {
		c.SetOwner(usr.Username)
		c.SetGroup(usr.Username)
	}

	// setup temporary directory for tests
	dir, err := ioutil.TempDir("", "test-cluster-dir")
	if err != nil {
		t.Fatal(err)
	}
	tempDirs = append(tempDirs, dir)
	c.SetVaultConfigPath(dir)
	c.SetDestination(dir + "/test")

	i = initInstanceToken(t, vaultDev, dir)

	return c, i
}

// Init kubernetes for testing
func initKubernetes(t *testing.T, vaultDev *vault_dev.VaultDev) *kubernetes.Kubernetes {
	k := kubernetes.New(vaultDev.Client())
	k.SetClusterID("test-cluster")

	if err := k.Ensure(); err != nil {
		t.Fatalf("error ensuring kubernetes: %s", err)
	}

	return k
}

// Start vault_dev for testing
func initVaultDev() *vault_dev.VaultDev {
	vaultDev := vault_dev.New()

	if err := vaultDev.Start(); err != nil {
		logrus.Fatalf("unable to initialise vault dev server for integration tests: %s", err)
	}

	return vaultDev
}

// Init instance token for testing
func initInstanceToken(t *testing.T, vaultDev *vault_dev.VaultDev, dir string) *instanceToken.InstanceToken {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	log := logrus.NewEntry(logger)

	i := instanceToken.New(vaultDev.Client(), log)
	i.SetRole("")
	i.SetClusterID("test-cluster")

	i.SetVaultConfigPath(dir)

	if _, err := os.Stat(i.InitTokenFilePath()); os.IsNotExist(err) {
		ifile, err := os.Create(i.InitTokenFilePath())
		if err != nil {
			t.Fatalf("%s", err)
		}
		defer ifile.Close()
	}

	_, err := os.Stat(i.TokenFilePath())
	if os.IsNotExist(err) {
		tfile, err := os.Create(i.TokenFilePath())
		if err != nil {
			t.Fatalf("%s", err)
		}
		defer tfile.Close()
	}

	i.WipeTokenFile(i.InitTokenFilePath())
	i.WipeTokenFile(i.TokenFilePath())

	return i
}