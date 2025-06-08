# About

This project provides a command-line interface to request an x509 certificate from Cumulocity. It is heavily re-using functionality from the go-client provided by [go-c8y](https://github.com/reubenmiller/go-c8y) library.

# Prerequisites

Following prerequisites are applying:

* The user needs `ROLE_DEVICE_CONTROL_ADMIN` permission in Cumulocity (the permission to register new Devices)

* The Cumulocity-CA feature being enabled in your tenant ([link](https://cumulocity.com/docs/device-certificate-authentication/certificate-authority/#prerequisites))

* CA Certificate must be present in your tenant ([link](https://cumulocity.com/docs/device-certificate-authentication/certificate-authority/#creating-a-ca-certificate-via-the-ui))

See [Cumulocity Certificate Authority](https://cumulocity.com/docs/device-certificate-authentication/certificate-authority/) for further info.

# Usage

The tool comes with following sub-commands:

* `registerUsingPassword` allowing to provide user-credentials (which will be used for automatic device-registration):

```
./c8y-get-certificate-from-ca registerUsingPassword \
  --device-id 'kobu-device-001' \                   # The associated device-identifier in Cumulocity
  --cumulocity-host 'https://iot.cumulocity.com' \  # Platform URL
  --cumulocity-tenant-id 't12345' \                 # Tenant ID of your Cumulocity Tenant
  --cumulocity-user 'john.doe' \                    # User for Certificate Request (needs to have ROLE_DEVICE_CONTROL_ADMIN permissson)
  --cumulocity-password 'superSecret1234'           # User password
```

* `registerUsingPoller`: This does not require user-credentials for enrollment. Instead, it will periodically poll for registration until a User created a matching Device Registration request in the target tenant.

```
./c8y-get-certificate-from-ca registerUsingPoller \
  --device-id 'kobu-device-001' \                   # The associated device-identifier in Cumulocity
  --cumulocity-host 'https://iot.cumulocity.com'    # Platform URL
```

* `renewCert`: Command is accepting current certificate and private-key and requests a new certificate with them.

```
./c8y-get-certificate-from-ca renewCert \
  --device-id 'kobu-device-001' \                   # The associated device-identifier in Cumulocity
  --cumulocity-host 'https://iot.cumulocity.com' \  # Platform URL
  --current-certificate ./c8y-certificate.pem \     # File path to certificate
  --private-key ./c8y-private-key.pem               # File path to private key
```

* `verifyCert`: Command accepts host, certificate and private key and tests if it's valid (by requesting an access token via HTTP). Exit Code 0 if valid, 1 if invalid.

```
./c8y-get-certificate-from-ca verifyCert \
  --cumulocity-host 'https://iot.cumulocity.com' \  # Platform URL
  --certificate ./c8y-certificate.pem               # File path to certificate
  --private-key ./c8y-private-key.pem               # File path to private key
```

* `getAccessToken`: Command accepts host, certificate and private key and responds with an access token obtained from Cumulocity

```
./c8y-get-certificate-from-ca verifyCert \
  --cumulocity-host 'https://iot.cumulocity.com' \  # Platform URL
  --certificate ./c8y-certificate.pem               # File path to certifictae
  --private-key ./c8y-private-key.pem               # Associated private key
```

# Miscellaneous

* The examples folder contains a script that can be used to configure the Cumulocity Edge deployment to use these certificates for Cloud Connectivity