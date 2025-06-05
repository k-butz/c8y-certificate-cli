package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

const fileNameTemplatePrivateKey = "c8y-private-key-%s.pem"
const fileNameTemplateCertificate = "c8y-certificate-%s.pem"

const exitCodePrerequisitesNotFulfilled int = 101
const exitCodeGeneralProcessingError int = 11

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

func readFromFile(fileName string) ([]byte, error) {
	b, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return b, nil
}

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
