package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

type CmdGroupGetAccessToken struct {
	C8yHost         string `long:"cumulocity-host" description:"Provide platform endpoint, e.g. 'https://iot.eu-latest.cumulocity.com'" required:"true"`
	CertificateFile string `long:"certificate" description:"File path to your certificate" required:"true"`
	PrivateKeyFile  string `long:"private-key" description:"File path to your private key" required:"true"`
}

var getAccessTokenCmdName = "getAccessToken"
var getAccessTokenCmdGroup CmdGroupGetAccessToken

func (g *CmdGroupGetAccessToken) Execute(args []string) error {
	slog.Info(fmt.Sprintf("Started %s with arguments: C8yHost=%s CertFile=%s PrivKeyFile=%s",
		renewCertCmdName, g.C8yHost, g.CertificateFile, g.PrivateKeyFile))

	certPEM, err := readFromFile(g.CertificateFile)
	if err != nil {
		slog.Error("Error when reading file. Exiting now.", "error", err, "fileName", g.CertificateFile)
		os.Exit(exitCodeGeneralProcessingError)
	}
	keyPem, err := readFromFile(g.PrivateKeyFile)
	if err != nil {
		slog.Error("Error when reading file. Exiting now.", "error", err, "fileName", g.PrivateKeyFile)
		os.Exit(exitCodeGeneralProcessingError)
	}
	clientCert, err := tls.X509KeyPair(certPEM, keyPem)
	if err != nil {
		slog.Error("Error while processing certificate and private key. Exiting now.", "error", err)
		os.Exit(exitCodeGeneralProcessingError)
	}
	client := c8y.NewClient(nil, g.C8yHost, "", "", "", false)
	token, tokenResp, err := client.DeviceEnrollment.RequestAccessToken(context.Background(), &clientCert, nil)
	if err != nil {
		slog.Error("Error while requesting access token", "error", err)
	}
	if tokenResp.Response.StatusCode != 200 {
		slog.Error("Unexpected response code while requesting first access token. Exiting now.", "expectedStatusCode", 200, "receivedStatusCode", tokenResp.Response.StatusCode)
		os.Exit(exitCodeGeneralProcessingError)
	}
	fmt.Println(fmt.Sprintf("Access Token obtained from %s:\n%s", client.BaseURL.Host, token.AccessToken))

	return nil
}
