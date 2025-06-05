#!/bin/sh

DEVICE_ID=kb-test-01
K8S_TLS_SECRET_NAME=c8y-cloud-tls-secret

# TODO: check if prerequisites are fulfilled (having jq, base64 and kubectl installed)

SECRET_VALUE=$(kubectl get secret $K8S_TLS_SECRET_NAME -n c8yedge -o json | jq -r '.data."tls.crt"' | base64 -d)
if [$? -ne 0]; then
    echo "Error while checking TLS secret $K8S_TLS_SECRET_NAME. Exiting now."
    exit 1
fi

# check if cert needs renewal
echo "$SECRET_VALUE" | openssl x509 -checkend 5184000 -noout
NEEDS_RENEWAL=$?
if [$NEEDS_RENEWAL -eq 1]; then
    echo "Found TLS cert. No renewal needed"
    exit 0
fi

./c8y-get-certificate-from-ca renewCert \
    --device-id "$DEVICE_ID" \
    --cumulocity-host $C8Y_HOST \
    --current-certificate "c8y-certificate-$DEVICE_ID.pem" \
    --private-key "c8y-private-key-$DEVICE_ID.pem"

# renewal succeeded, now re-create the TLS secret with new PEM files
if [$? -eq 0]; then
    kubectl delete secret -n c8yedge $K8S_TLS_SECRET_NAME
    kubectl create secret tls $TLS_SECRET_NAME -n c8yedge --cert c8y-certificate-$DEVICE_ID.pem --key c8y-private-key-$DEVICE_ID.pem
fi
