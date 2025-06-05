package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

type CmdGroupEnrollmentPoller struct {
	C8yHost  string `long:"cumulocity-host" description:"Provide platform endpoint, e.g. 'https://iot.eu-latest.cumulocity.com'" required:"true"`
	DeviceId string `long:"device-id" description:"Provide identifier for your Cloud device, e.g. 'kobu-edge-01'. Free text but needs to be unique." required:"true"`
	Otp      string `long:"one-time-password" description:"One time password to be used for enrollment. Optional (auto-created when missing)" required:"false"`
}

var regUsingPollerCmdName = "registerUsingPoller"
var regUsingPollerCmdGroup CmdGroupEnrollmentPoller

func (g *CmdGroupEnrollmentPoller) Execute(args []string) error {
	slog.Info(fmt.Sprintf("Started %s with arguments: C8yHost=%s C8yTenantId=%s DeviceId=%s",
		regUsingPollerCmdName, g.C8yHost, "", g.DeviceId))

	client := c8y.NewClient(nil, g.C8yHost, "", "", "", false)

	keyPem, err := certutil.MakeEllipticPrivateKeyPEM()
	if err != nil {
		slog.Error("Error while creating private key. Exiting now.", "error", err)
		os.Exit(exitCodeGeneralProcessingError)
	}

	key, err := certutil.ParsePrivateKeyPEM(keyPem)
	if err != nil {
		slog.Error("Error while parsing private key. Exiting now.", "error", err)
		os.Exit(exitCodeGeneralProcessingError)
	}

	slog.Info("Starting device enrollment", "externalId", g.DeviceId)

	csr, err := client.DeviceEnrollment.CreateCertificateSigningRequest(g.DeviceId, key)
	if err != nil {
		slog.Error("Error while creating Certificate signing request. Exiting now.", "error", err)
		os.Exit(exitCodeGeneralProcessingError)
	}

	ctx := c8y.NewSilentLoggerContext(context.Background())

	otp := g.Otp
	if len(otp) == 0 {
		slog.Info("No one-time-password provided. Generating it...", "otp", otp)
		otp, err = client.DeviceEnrollment.GenerateOneTimePassword()
		if err != nil {
			slog.Error("Error while generating one time password. Exiting now.", "error", err)
			os.Exit(exitCodeGeneralProcessingError)
		}
	}

	result := <-client.DeviceEnrollment.PollEnroll(ctx, c8y.DeviceEnrollmentOption{
		ExternalID:      g.DeviceId,
		OneTimePassword: otp,
		InitDelay:       2 * time.Second,
		Interval:        5 * time.Second,
		Timeout:         10 * time.Minute,
		Banner: &c8y.DeviceEnrollmentBannerOptions{
			Enable:     true,
			ShowQRCode: true,
			ShowURL:    true,
		},
		CertificateSigningRequest: csr,
		OnProgressBefore: func() {
			fmt.Fprintf(os.Stderr, "\rTrying to download certificate: ")
		},
		OnProgressError: func(r *c8y.Response, err error) {
			fmt.Fprintf(os.Stderr, "WAITING (last statusCode=%s, time=%s)\n", r.Status(), time.Now().Format(time.RFC3339))
		},
	})
	if result.Err != nil {
		slog.Error("Failed to download the device certificate")
		os.Exit(1)
	}
	slog.Info("Successfully download the device certificate")

	cert := result.Certificate
	certPEM := certutil.MarshalCertificateToPEM(cert.Raw)
	if len(string(certPEM)) == 0 {
		slog.Error("Error while converting certificate from []byte to PEM format", "error", "PEM length is 0")
		os.Exit(exitCodeGeneralProcessingError)
	}

	privateKeyFileName := fmt.Sprintf(fileNameTemplatePrivateKey, g.DeviceId)
	certFileName := fmt.Sprintf(fileNameTemplateCertificate, g.DeviceId)
	writeToFile(string(keyPem), privateKeyFileName)
	writeToFile(string(certPEM), certFileName)

	slog.Info(fmt.Sprintf("Certificate retrieval succeeded. Placed files '%s' and '%s' in current working directory.",
		privateKeyFileName, certFileName))

	return nil
}
