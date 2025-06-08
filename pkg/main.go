package main

import (
	"os"

	"github.com/jessevdk/go-flags"
)

func init() {
	parser.AddCommand(regUsingPassCmdGroupName,
		"Register a device using password",
		"This command will create private key, CSR, a registration request in the platform (using provided user credentials) and downloads the matching certificate",
		&regUsingPassCmdGroup)

	parser.AddCommand(regUsingPollerCmdName,
		"Register a device using enrollment poller",
		"This command will create private key, CSR and starts polling for device credentials. Once a user does the registration, the certificate will be downloaded",
		&regUsingPollerCmdGroup)

	parser.AddCommand(renewCertCmdName,
		"Renew certificate",
		"This command uses an existing certifidate and requests/downloads a new one",
		&renewCertCmdGroup)

	parser.AddCommand(getAccessTokenCmdName,
		"Get Access Token",
		"This command accepts private key and certificate and requests an Access Token via Cumulocitys HTTP/REST API",
		&getAccessTokenCmdGroup)

	parser.AddCommand(verifyCertificateCmdName,
		"Verify certificate",
		"This command accepts private key and certificate and tests if it's valid (by requesting access token via HTTP)",
		&verifyCertificateCmdGroup)

	parser.AddCommand(versionCmdName,
		"Version",
		"This command tells about the current tool version",
		&versionCmdGroup)
}

var parser = flags.NewParser(nil, flags.Default)

func main() {
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}
}
