package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog"
)

var Name, Namespace string
var ResourceConfigDir string

func main() {

	AddBuildResourceConfigFlags(cmd)
	if err := cmd.Execute(); err != nil {
		klog.Fatal(err)
	}
}

func RunMain(cmd *cobra.Command, args []string) {
	cmd.Help()
}

var cmd = &cobra.Command{
	Use:   "config",
	Short: "Create certificates for configuring APIService.",
	Long:  `Create certificates for configuring APIService`,
	Example: `
# Generates self-signed certificates into the config/ directory for running the apiserver.
--name nameofservice --namespace mysystemnamespace
`,
	Run: RunBuildResourceConfig,
}

func AddBuildResourceConfigFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&Name, "name", "", "")
	cmd.Flags().StringVar(&Namespace, "namespace", "", "")
	cmd.Flags().StringVar(&ResourceConfigDir, "output", "config", "directory to output resourceconfig")
}

func RunBuildResourceConfig(cmd *cobra.Command, args []string) {
	if len(Name) == 0 {
		klog.Fatalf("must specify --name")
	}
	if len(Namespace) == 0 {
		klog.Fatalf("must specify --namespace")
	}

	generateAPIServerCerts(ResourceConfigDir)
	generateDBCerts(ResourceConfigDir)
}

func generateAPIServerCerts(resourceConfigDir string) {
	dir := filepath.Join(resourceConfigDir, "apiserver")
	generateCerts(dir)
}

func generateDBCerts(resourceConfigDir string) {
	dir := filepath.Join(resourceConfigDir, "db")
	generateCerts(dir)
}

func generateCerts(dir string) {
	createCerts(dir)
	ClientKey := getBase64(filepath.Join(dir, "cert.key"))
	WriteStringToFile(filepath.Join(dir, "key.txt"), ClientKey)

	CACert := getBase64(filepath.Join(dir, "cacrt.crt"))
	WriteStringToFile(filepath.Join(dir, "cacrt.txt"), CACert)

	ClientCert := getBase64(filepath.Join(dir, "cert.crt"))
	WriteStringToFile(filepath.Join(dir, "cert.txt"), ClientCert)
}

func WriteStringToFile(filepath, s string) error {
	fo, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fo.Close()

	_, err = io.Copy(fo, strings.NewReader(s))
	if err != nil {
		return err
	}

	return nil
}

func getBase64(file string) string {

	buff := bytes.Buffer{}
	enc := base64.NewEncoder(base64.StdEncoding, &buff)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		klog.Fatalf("Could not read file %s: %v", file, err)
	}

	_, err = enc.Write(data)
	if err != nil {
		klog.Fatalf("Could not write bytes: %v", err)
	}
	enc.Close()
	return buff.String()

}

func createCerts(dir string) {
	os.MkdirAll(dir, 0700)

	if _, err := os.Stat(filepath.Join(dir, "cacrt.crt")); os.IsNotExist(err) {
		DoCmd("openssl", "req", "-x509",
			"-newkey", "rsa:2048",
			"-keyout", filepath.Join(dir, "cacrt.key"),
			"-out", filepath.Join(dir, "cacrt.crt"),
			"-days", "365",
			"-nodes",
			"-subj", fmt.Sprintf("/C=un/ST=st/L=l/O=o/OU=ou/CN=%s-certificate-authority", Name),
		)
	} else {
		klog.Infof("Skipping generate CA cert.  File already exists.")
	}

	if _, err := os.Stat(filepath.Join(dir, "cert.csr")); os.IsNotExist(err) {
		// Use <service-Name>.<Namespace>.svc as the domain Name for the certificate
		DoCmd("openssl", "req",
			"-out", filepath.Join(dir, "cert.csr"),
			"-new",
			"-newkey", "rsa:2048",
			"-nodes",
			"-keyout", filepath.Join(dir, "cert.key"),
			"-subj", fmt.Sprintf("/C=un/ST=st/L=l/O=o/OU=ou/CN=%s.%s.svc", Name, Namespace),
		)
	} else {
		klog.Infof("Skipping generate cert csr.  File already exists.")
	}

	if _, err := os.Stat(filepath.Join(dir, "cert.crt")); os.IsNotExist(err) {
		DoCmd("openssl", "x509", "-req",
			"-days", "365",
			"-in", filepath.Join(dir, "cert.csr"),
			"-CA", filepath.Join(dir, "cacrt.crt"),
			"-CAkey", filepath.Join(dir, "cacrt.key"),
			"-CAcreateserial",
			"-out", filepath.Join(dir, "cert.crt"),
		)
	} else {
		klog.Infof("Skipping signing cert crt.  File already exists.")
	}
}

func DoCmd(cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	klog.Infof("%s", strings.Join(c.Args, " "))
	err := c.Run()
	if err != nil {
		klog.Fatalf("command failed %v", err)
	}
}
