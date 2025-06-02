package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

type cmdLineArgInput struct {
	c8yHost     string
	c8yTenant   string
	c8yUser     string
	c8yPassword string
	deviceId    string
}

const c8yHostCmdLineFlag = "cumulocity-host"
const c8yTenantCmdLineFlag = "cumulocity-tenant-id"
const c8yUserCmdLineFlag = "cumulocity-user"
const c8yPasswordCmdLineFlag = "cumulocity-password"
const deviceIdCmdLineFlag = "device-id"

const fileNameTemplatePrivateKey = "c8y-private-key-%s.pem"
const fileNameTemplateCertificate = "c8y-certificate-%s.pem"

func main() {
	cmdLineArgInput := &cmdLineArgInput{}
	flag.StringVar(&cmdLineArgInput.c8yHost, c8yHostCmdLineFlag, "", "Provide platform endpoint, e.g. 'https://iot.eu-latest.cumulocity.com'")
	flag.StringVar(&cmdLineArgInput.c8yTenant, c8yTenantCmdLineFlag, "", "Provide platform tenand id, e.g. 't4009123'")
	flag.StringVar(&cmdLineArgInput.c8yUser, c8yUserCmdLineFlag, "", "Provide your platform user, e.g. 'john.doe@example.org'")
	flag.StringVar(&cmdLineArgInput.c8yPassword, c8yPasswordCmdLineFlag, "", "Provide your platform users password, e.g. 'aVerySecretPass1337'")
	flag.StringVar(&cmdLineArgInput.deviceId, deviceIdCmdLineFlag, "", "Provide identifier for your Cloud device, e.g. 'kobu-edge-01'. Free text but needs to be unique.")
	slog.Info("Processing provided command line arguments")
	flag.Parse()
	if e := cmdLineArgInput.validateInput(); e != nil {
		slog.Error("Provided command line arguments are invalid.", "error", e)
		os.Exit(100)
	}

	client := c8y.NewClient(nil, cmdLineArgInput.c8yHost, cmdLineArgInput.c8yTenant, cmdLineArgInput.c8yUser, cmdLineArgInput.c8yPassword, false)
	currentTenant, _, e := client.Tenant.GetCurrentTenant(context.TODO())
	if e != nil {
		slog.Error("Error while retrieving current tenant. Did you set the expected environment variables? Exiting now.", "error", e)
		os.Exit(90)
	}
	domainName := currentTenant.DomainName
	slog.Info("Starting routine in tenant " + domainName)

	slog.Info("Testing user for having the required permissions")
	if e := checkForRequiredRoles(client, "ROLE_DEVICE_CONTROL_ADMIN"); e != nil {
		slog.Error("Error while checking User permissions. Exiting now.", "error", e)
		os.Exit(91)
	}

	slog.Info("Testing if CA Certificate is existing")
	if _, e = client.CertificateAuthority.Get(context.TODO()); e != nil {
		slog.Error("Error while requesting certificate. Is the CA certificate created in "+domainName+"? Exiting now.", "error", e)
		os.Exit(92)
	}

	deviceID := cmdLineArgInput.deviceId
	otp, e := client.DeviceEnrollment.GenerateOneTimePassword()
	if e != nil {
		slog.Error("Error while creating one time password", "error", e, "deviceID", deviceID)
		os.Exit(2)
	}

	slog.Info("Creating bulk registration request for device-id", "deviceID", deviceID)
	if e = createBulkRegistrationRequest(deviceID, otp, client); e != nil {
		slog.Error("Error while creating bulk registration request. Exiting now.", "error", e, "deviceID", deviceID)
		os.Exit(3)
	}

	slog.Info("Creating private key for device-id", "deviceID", deviceID)
	keyPem, e := certutil.MakeEllipticPrivateKeyPEM()
	if e != nil {
		slog.Error("Error wile creating private key. Exiting now.", "error", e, "deviceID", deviceID)
		os.Exit(4)
	}

	slog.Info("Parsing Private Key from PEM", "deviceID", deviceID)
	key, e := certutil.ParsePrivateKeyPEM(keyPem)
	if e != nil {
		slog.Error("Error wile parsing private key. Exiting now.", "error", e, "deviceID", deviceID)
		os.Exit(5)
	}

	slog.Info("Creating certificate signing request", "deviceID", deviceID)
	csr, e := CreateCertificateSigningRequest(deviceID, key)
	if e != nil {
		slog.Error("Error while creating CSR. Exiting now.", "error", e)
		os.Exit(6)
	}

	slog.Info("Enrolling Device", "deviceID", deviceID)
	certPEM, e := enrollDevice(client, deviceID, otp, csr, 5)
	if e != nil {
		slog.Error("Error while enrolling device", "error", e)
		os.Exit(7)
	}

	slog.Info("Extracting Private/Public Keypair from PEM")
	clientCert, e := tls.X509KeyPair(certPEM, keyPem)
	if e != nil {
		slog.Error("Error while extracting private/public keypair from PEM. Exiting now.", "deviceID", deviceID, "error", e)
		os.Exit(8)
	}

	// requesting an access token to test if the .pems are working well
	slog.Info("Request access token for client certificate")
	if e := verifyPlatformAccessWithCert(client, clientCert); e != nil {
		slog.Error("Error while getting an access token with provided certificate", "error", e)
		os.Exit(9)
	}

	privateKeyFileName := fmt.Sprintf(fileNameTemplatePrivateKey, deviceID)
	certFileName := fmt.Sprintf(fileNameTemplateCertificate, deviceID)
	writeToFile(string(keyPem), privateKeyFileName)
	writeToFile(string(certPEM), certFileName)
	slog.Info(fmt.Sprintf("Certificate retrieval has been successful :) Placed files '%s' and '%s' in current working directory.", privateKeyFileName, certFileName))

	os.Exit(0)
}

func (cmdLineArgInput *cmdLineArgInput) validateInput() error {
	if len(cmdLineArgInput.deviceId) == 0 {
		return errors.New(fmt.Sprintf("Missing input for %s argument", deviceIdCmdLineFlag))
	}
	if len(cmdLineArgInput.c8yHost) == 0 {
		return errors.New(fmt.Sprintf("Missing input for %s argument", c8yHostCmdLineFlag))
	}
	if len(cmdLineArgInput.c8yTenant) == 0 {
		return errors.New(fmt.Sprintf("Missing input for %s argument", c8yTenantCmdLineFlag))
	}
	if len(cmdLineArgInput.c8yUser) == 0 {
		return errors.New(fmt.Sprintf("Missing input for %s argument", c8yUserCmdLineFlag))
	}
	if len(cmdLineArgInput.c8yPassword) == 0 {
		return errors.New(fmt.Sprintf("Missing input for %s argument", c8yPasswordCmdLineFlag))
	}
	return nil
}

func CreateCertificateSigningRequest(deviceID string, key interface{}) (*x509.CertificateRequest, error) {
	csr, e := certutil.CreateCertificateSigningRequest(deviceID, key)
	if e != nil {
		return &x509.CertificateRequest{}, e
	}
	if csr.Subject.CommonName != deviceID {
		return &x509.CertificateRequest{}, errors.New("Common name field of CSR does not match with device id")
	}
	return csr, nil
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

// Requests an access token from the platform via HTTP. Returns error in case request does not succeed or access token is invalid.
func verifyPlatformAccessWithCert(client *c8y.Client, clientCert tls.Certificate) error {
	token, tokenResp, e := client.DeviceEnrollment.RequestAccessToken(context.TODO(), &clientCert, nil)
	if e != nil {
		return errors.New("Error while requesting access token with client certificate")
	}
	statusCode := tokenResp.Response.StatusCode
	if statusCode != 200 {
		return errors.New(fmt.Sprintf("Invalid status code received while requesting access token with client certificate (expected 200, received %d)", statusCode))
	}
	if len(token.AccessToken) == 0 {
		return errors.New("Received an access token from platform but it's empty")
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

func writeToFile(content string, fileName string) error {
	f, err := os.Create(fileName)
	defer f.Close()
	if err != nil {
		return err
	}

	if _, err = f.WriteString(content); err != nil {
		return err
	}

	return nil
}
