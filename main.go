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
	certs := [2]string{"apiserver", "db"}
	for _, cert := range certs {
		dir := filepath.Join(ResourceConfigDir, cert)
		createCerts(dir)
		ClientKey := getBase64(filepath.Join(dir, "apiserver.key"))
		WriteStringToFile(filepath.Join(dir, "key.txt"), ClientKey)

		CACert := getBase64(filepath.Join(dir, "apiserver_ca.crt"))
		WriteStringToFile(filepath.Join(dir, "cacrt.txt"), CACert)

		ClientCert := getBase64(filepath.Join(dir, "apiserver.crt"))
		WriteStringToFile(filepath.Join(dir, "crt.txt"), ClientCert)
	}

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
	//out, err := exec.Command("bash", "-c",
	//	fmt.Sprintf("base64 %s | awk 'BEGIN{ORS=\"\";} {print}'", file)).CombinedOutput()
	//if err != nil {
	//	klog.Fatalf("Could not base64 encode file: %v", err)
	//}

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

	//if string(out) != buff.String() {
	//	fmt.Printf("\nNot Equal\n")
	//}
	//
	//return string(out)
}

func createCerts(dir string) {
	os.MkdirAll(dir, 0700)

	if _, err := os.Stat(filepath.Join(dir, "apiserver_ca.crt")); os.IsNotExist(err) {
		DoCmd("openssl", "req", "-x509",
			"-newkey", "rsa:2048",
			"-keyout", filepath.Join(dir, "apiserver_ca.key"),
			"-out", filepath.Join(dir, "apiserver_ca.crt"),
			"-days", "365",
			"-nodes",
			"-subj", fmt.Sprintf("/C=un/ST=st/L=l/O=o/OU=ou/CN=%s-certificate-authority", Name),
		)
	} else {
		klog.Infof("Skipping generate CA cert.  File already exists.")
	}

	if _, err := os.Stat(filepath.Join(dir, "apiserver.csr")); os.IsNotExist(err) {
		// Use <service-Name>.<Namespace>.svc as the domain Name for the certificate
		DoCmd("openssl", "req",
			"-out", filepath.Join(dir, "apiserver.csr"),
			"-new",
			"-newkey", "rsa:2048",
			"-nodes",
			"-keyout", filepath.Join(dir, "apiserver.key"),
			"-subj", fmt.Sprintf("/C=un/ST=st/L=l/O=o/OU=ou/CN=%s.%s.svc", Name, Namespace),
		)
	} else {
		klog.Infof("Skipping generate apiserver csr.  File already exists.")
	}

	if _, err := os.Stat(filepath.Join(dir, "apiserver.crt")); os.IsNotExist(err) {
		DoCmd("openssl", "x509", "-req",
			"-days", "365",
			"-in", filepath.Join(dir, "apiserver.csr"),
			"-CA", filepath.Join(dir, "apiserver_ca.crt"),
			"-CAkey", filepath.Join(dir, "apiserver_ca.key"),
			"-CAcreateserial",
			"-out", filepath.Join(dir, "apiserver.crt"),
		)
	} else {
		klog.Infof("Skipping signing apiserver crt.  File already exists.")
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
