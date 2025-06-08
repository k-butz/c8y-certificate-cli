package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

type CmdGroupVerifyCertificate struct {
	C8yHost         string `long:"cumulocity-host" description:"Provide platform endpoint, e.g. 'https://iot.eu-latest.cumulocity.com'" required:"true"`
	CertificateFile string `long:"certificate" description:"File path to your certificate" required:"true"`
	PrivateKeyFile  string `long:"private-key" description:"File path to your private key" required:"true"`
}

var verifyCertificateCmdName = "verifyCert"
var verifyCertificateCmdGroup CmdGroupVerifyCertificate

func (g *CmdGroupVerifyCertificate) Execute(args []string) error {
	certPEM, err := readFromFile(g.CertificateFile)
	if err != nil {
		errMessage := fmt.Sprintf("Error when reading file. Error = %s. File = %s", err.Error(), g.CertificateFile)
		exitWithErr(errMessage)
	}
	keyPem, err := readFromFile(g.PrivateKeyFile)
	if err != nil {
		errMessage := fmt.Sprintf("Error when reading file. Error = %s. File = %s", err.Error(), g.PrivateKeyFile)
		exitWithErr(errMessage)
	}
	clientCert, err := tls.X509KeyPair(certPEM, keyPem)
	if err != nil {
		errMessage := fmt.Sprintf("Error while processing certificate and private key. Error = %s.", err.Error())
		exitWithErr(errMessage)
	}
	client := c8y.NewClient(nil, g.C8yHost, "", "", "", false)
	_, tokenResp, err := client.DeviceEnrollment.RequestAccessToken(context.Background(), &clientCert, nil)
	if err != nil {
		errMessage := fmt.Sprintf("Error while requesting access token. Error = %s.", err.Error())
		exitWithErr(errMessage)
	}
	if tokenResp.Response.StatusCode != 200 {
		errMessage := fmt.Sprintf("Unexpected response code while requesting first access token. Expected 200, received %d",
			tokenResp.Response.StatusCode)
		exitWithErr(errMessage)
	}
	fmt.Println("Verification result: OK")

	return nil
}

func exitWithErr(errorMessage string) {
	fmt.Println("Verification result: NOT_OK")
	fmt.Println("Reason: " + errorMessage)
	os.Exit(1)
}
