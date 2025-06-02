# About

This project provides a command-line interface to request an x509 certificate from Cumulocity. 

# Prerequisites

Following prerequisites are applying:

* The user needs `ROLE_DEVICE_CONTROL_ADMIN` permission in Cumulocity (the permission to register new Devices)

* The Cumulocity-CA feature being enabled in your tenant ([link](https://cumulocity.com/docs/device-certificate-authentication/certificate-authority/#prerequisites))

* CA Certificate must be present in your tenant ([link](https://cumulocity.com/docs/device-certificate-authentication/certificate-authority/#creating-a-ca-certificate-via-the-ui))

See [Cumulocity Certificate Authority](https://cumulocity.com/docs/device-certificate-authentication/certificate-authority/) for further info.

# Usage

The tool can be used as follows:

```
./c8y-get-certificate-from-ca \
  --device-id 'kobu-device-001' \                   # the associated device-identifier in Cumulocity
  --cumulocity-host 'https://iot.cumulocity.com' \  # Platform URL
  --cumulocity-tenant-id 't12345' \                 # Tenant ID of your Cumulocity Tenant
  --cumulocity-user 'john.doe' \                    # User for Certificate Request (needs to have ROLE_DEVICE_CONTROL_ADMIN permissson)
  --cumulocity-password 'superSecret1234'           # User password
```

The toool will place `c8y-private-key-{device-id}.pem` and `c8y-certificate-{device-id}.pem` to your current working directory. The certificate can now be used to communicate with the platform via MQTTS and HTTPS. 

# Miscellaneous

* The examples folder contains a script that can be used to configure the Cumulocity Edge deployment to use these certificates for Cloud Connectivity