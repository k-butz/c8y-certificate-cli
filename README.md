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

* `registerUsingPassword`: Allowing to provide user-credentials that will be used for automatic device-registration:

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
  --cumulocity-host 'https://iot.cumulocity.com' \  # Platform URL
```

* `renewCert`: Command is accepting current certificate and private-key and requests a new certificate with them.

```
./c8y-get-certificate-from-ca renewCert \
  --device-id 'kobu-device-001' \                   # The associated device-identifier in Cumulocity
  --cumulocity-host 'https://iot.cumulocity.com' \  # Platform URL
  --current-certificate ./c8y-certificate.pem       # Current certificate (which needs to be still valid)
  --private-key ./c8y-private-key.pem               # Associated private key
```

The tool will place `c8y-private-key-{device-id}.pem` and `c8y-certificate-{device-id}.pem` to your current working directory. The certificate can now be used to communicate with the platform via MQTTS and HTTPS. 

# Miscellaneous

* The examples folder contains a script that can be used to configure the Cumulocity Edge deployment to use these certificates for Cloud Connectivity