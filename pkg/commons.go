package main

import (
	"os"
)

const fileNameTemplatePrivateKey = "c8y-private-key-%s.pem"
const fileNameTemplateCertificate = "c8y-certificate-%s.pem"

const exitCodePrerequisitesNotFulfilled int = 101
const exitCodeGeneralProcessingError int = 1

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
