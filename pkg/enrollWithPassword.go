package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

type CmdGroupRegisterUsingPassword struct {
	C8yHost     string `long:"cumulocity-host" description:"Provide platform endpoint, e.g. 'https://iot.eu-latest.cumulocity.com'" required:"true"`
	C8yTenantId string `long:"cumulocity-tenant-id" description:"Provide platform tenand id, e.g. 't4009123'" required:"true"`
	DeviceId    string `long:"device-id" description:"Provide identifier for your Cloud device, e.g. 'kobu-edge-01'. Free text but needs to be unique." required:"true"`
	C8yUser     string `long:"cumulocity-user" description:"Provide your platform user, e.g. 'john.doe@example.org'" required:"true"`
	C8yPassword string `long:"cumulocity-password" description:"Provide your platform users password, e.g. 'aVerySecretPass1337'" required:"true"`
}

var regUsingPassCmdGroupName = "registerUsingPassword"
var regUsingPassCmdGroup CmdGroupRegisterUsingPassword

func (g *CmdGroupRegisterUsingPassword) Execute(args []string) error {
	slog.Info(fmt.Sprintf("Started with arguments: C8yHost=%s C8yTenantId=%s C8yUser=%s C8yPassword=%s DeviceId=%s",
		g.C8yHost, g.C8yTenantId, g.C8yUser, "{obfuscated}", g.DeviceId))

	client := c8y.NewClient(nil, g.C8yHost, g.C8yTenantId, g.C8yUser, g.C8yPassword, false)
	currentTenant, _, e := client.Tenant.GetCurrentTenant(context.TODO())
	if e != nil {
		slog.Error("Error while retrieving current tenant. Did you set the expected environment variables? Exiting now.", "error", e)
		os.Exit(exitCodePrerequisitesNotFulfilled)
	}
	domainName := currentTenant.DomainName
	slog.Info("Starting routine in tenant " + domainName)

	slog.Info("Testing user for having the required permissions")
	if e := checkForRequiredRoles(client, "ROLE_DEVICE_CONTROL_ADMIN"); e != nil {
		slog.Error("Error while checking User permissions. Exiting now.", "error", e)
		os.Exit(exitCodePrerequisitesNotFulfilled)
	}

	slog.Info("Testing if CA Certificate is existing")
	if _, e = client.CertificateAuthority.Get(context.TODO()); e != nil {
		slog.Error("Error while requesting certificate. Is the CA certificate created in "+domainName+"? Exiting now.", "error", e)
		os.Exit(exitCodePrerequisitesNotFulfilled)
	}

	deviceID := g.DeviceId
	otp, e := client.DeviceEnrollment.GenerateOneTimePassword()
	if e != nil {
		slog.Error("Error while creating one time password", "error", e, "deviceID", deviceID)
		os.Exit(exitCodeGeneralProcessingError)
	}

	slog.Info("Creating bulk registration request for device-id", "deviceID", deviceID)
	if e = createBulkRegistrationRequest(deviceID, otp, client); e != nil {
		slog.Error("Error while creating bulk registration request. Exiting now.", "error", e, "deviceID", deviceID)
		os.Exit(exitCodeGeneralProcessingError)
	}

	slog.Info("Creating private key for device-id", "deviceID", deviceID)
	keyPem, e := certutil.MakeEllipticPrivateKeyPEM()
	if e != nil {
		slog.Error("Error wile creating private key. Exiting now.", "error", e, "deviceID", deviceID)
		os.Exit(exitCodeGeneralProcessingError)
	}

	slog.Info("Parsing Private Key from PEM", "deviceID", deviceID)
	key, e := certutil.ParsePrivateKeyPEM(keyPem)
	if e != nil {
		slog.Error("Error wile parsing private key. Exiting now.", "error", e, "deviceID", deviceID)
		os.Exit(exitCodeGeneralProcessingError)
	}

	slog.Info("Creating certificate signing request", "deviceID", deviceID)
	csr, e := createCertificateSigningRequest(deviceID, key)
	if e != nil {
		slog.Error("Error while creating CSR. Exiting now.", "error", e)
		os.Exit(exitCodeGeneralProcessingError)
	}

	slog.Info("Enrolling Device", "deviceID", deviceID)
	certPEM, e := enrollDevice(client, deviceID, otp, csr, 5)
	if e != nil {
		slog.Error("Error while enrolling device", "error", e)
		os.Exit(exitCodeGeneralProcessingError)
	}

	slog.Info("Extracting Private/Public Keypair from PEM")
	clientCert, e := tls.X509KeyPair(certPEM, keyPem)
	if e != nil {
		slog.Error("Error while extracting private/public keypair from PEM. Exiting now.", "deviceID", deviceID, "error", e)
		os.Exit(exitCodeGeneralProcessingError)
	}

	// requesting an access token to test if the .pems are working well
	slog.Info("Request access token for client certificate")
	if e := verifyPlatformAccessWithCert(client, clientCert); e != nil {
		slog.Error("Error while getting an access token with provided certificate", "error", e)
		os.Exit(exitCodeGeneralProcessingError)
	}

	privateKeyFileName := fmt.Sprintf(fileNameTemplatePrivateKey, deviceID)
	certFileName := fmt.Sprintf(fileNameTemplateCertificate, deviceID)
	writeToFile(string(keyPem), privateKeyFileName)
	writeToFile(string(certPEM), certFileName)
	slog.Info(fmt.Sprintf("Certificate retrieval succeeded. Placed files '%s' and '%s' in current working directory.", privateKeyFileName, certFileName))

	return nil
}

func enrollDevice(client *c8y.Client, deviceID string, otp string, csr *x509.CertificateRequest, maxRetries int) ([]byte, error) {
	attempt := 0
	var cert *x509.Certificate
	for {
		attempt++
		var resp *c8y.Response
		var e error
		cert, resp, e = client.DeviceEnrollment.Enroll(context.TODO(), deviceID, otp, csr)
		if e == nil && resp.Response.StatusCode == 200 {
			slog.Info("Device enrollment request succeeded", "deviceID", deviceID, "attempt", attempt, "statusCode", resp.Response.StatusCode)
			slog.Info("Marshal certificate to PEM")
			certPEM := certutil.MarshalCertificateToPEM(cert.Raw)
			return certPEM, nil
		} else {
			if attempt == maxRetries {
				return nil, errors.New("Giving up device enrollment request after 5 retrials")
			}
			slog.Warn("Error while device enrollment. Retrying in 3 seconds.", "deviceID", deviceID, "statusCode", resp.Response.StatusCode, "error", e, "attempt", attempt)
			time.Sleep(time.Second * 3)
		}
	}

}

// Requests users current permissions and checks if provided requiredRole is part of it. Returns error if not.
func checkForRequiredRoles(client *c8y.Client, requiredRole string) error {
	currentUser, _, e := client.User.GetCurrentUser(context.TODO())
	if e != nil {
		return errors.New("Error while retrieving users permissions: " + e.Error())
	}
	containsRequiredRole := false
	for _, value := range currentUser.EffectiveRoles {
		if strings.ToUpper(value.Name) == strings.ToUpper(requiredRole) {
			containsRequiredRole = true
			break
		}
	}
	if !containsRequiredRole {
		return errors.New("User does not have the required permission " + requiredRole)
	}
	return nil
}

func createBulkRegistrationRequest(deviceID string, otp string, client *c8y.Client) error {
	csvContents := bytes.NewBufferString("")
	csvWriter := csv.NewWriter(csvContents)
	csvWriter.Comma = '\t'
	_ = csvWriter.Write([]string{
		"ID",
		"AUTH_TYPE",
		"ENROLLMENT_OTP",
		"NAME",
		"TYPE",
		"IDTYPE",
		"com_cumulocity_model_Agent.active",
	})
	_ = csvWriter.Write([]string{
		deviceID,
		"CERTIFICATES",
		otp,
		deviceID,
		"test_ci_reg",
		"c8y_Serial",
		"true",
	})
	csvWriter.Flush()
	_, resp, err := client.DeviceCredentials.CreateBulk(context.TODO(), csvContents)
	slog.Info("Response status code for bulk registration request", "deviceID", deviceID, "statusCode", resp.Response.StatusCode)
	if err != nil {
		return err
	}
	if resp.Response.StatusCode != 201 {
		return errors.New(fmt.Sprintf("Invalid response status code %d from platform. Expected 201.", resp.Response.StatusCode))
	}
	return nil
}
