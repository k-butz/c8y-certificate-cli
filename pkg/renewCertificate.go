package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

type CmdGroupRenewCert struct {
	C8yHost         string `long:"cumulocity-host" description:"Provide platform endpoint, e.g. 'https://iot.eu-latest.cumulocity.com'" required:"true"`
	DeviceId        string `long:"device-id" description:"The associated device-id from your platform device" required:"true"`
	CertificateFile string `long:"current-certificate" description:"File path to your certificate pem" required:"true"`
	PrivateKeyFile  string `long:"private-key" description:"File path to your private key pem" required:"true"`
}

var renewCertCmdName = "renewCert"
var renewCertCmdGroup CmdGroupRenewCert

func (g *CmdGroupRenewCert) Execute(args []string) error {
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

	key, err := certutil.ParsePrivateKeyPEM(keyPem)
	if err != nil {
		slog.Error("Error while parsing private key. Exiting now.", "error", err)
		os.Exit(exitCodeGeneralProcessingError)
	}
	csr, err := client.DeviceEnrollment.CreateCertificateSigningRequest(g.DeviceId, key)
	if err != nil {
		slog.Error("Error while creating certificate signing request. Exiting now.", "error", err)
		os.Exit(exitCodeGeneralProcessingError)
	}
	cert, resp, err := client.DeviceEnrollment.ReEnroll(context.Background(), c8y.ReEnrollOptions{
		Token: token.AccessToken,
		CSR:   csr,
	})
	if err != nil {
		slog.Error("Error while sending re-enrollment request. Exiting now.", "error", err)
		os.Exit(exitCodeGeneralProcessingError)
	}
	if resp.Response.StatusCode != 200 {
		slog.Error("Unexpected response code for re-enrollment request. Exiting now.", "expectedStatusCode", 200, "receivedStatusCode", resp.Response.StatusCode)
		os.Exit(exitCodeGeneralProcessingError)
	}

	newCertPEM := certutil.MarshalCertificateToPEM(cert.Raw)
	if len(string(newCertPEM)) == 0 {
		slog.Error("Error while converting certificate from []byte to PEM format", "error", "PEM length is 0")
		os.Exit(exitCodeGeneralProcessingError)
	}

	certFileName := fmt.Sprintf(fileNameTemplateCertificate, g.DeviceId+".new")
	writeToFile(string(newCertPEM), certFileName)

	slog.Info(fmt.Sprintf("Certificate renewal succeeded. Placed file '%s' in current working directory.",
		certFileName))

	return nil
}
