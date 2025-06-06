#!/bin/sh

DEVICE_ID=kb-test-01
K8S_TLS_SECRET_NAME=c8y-cloud-tls-secret

log(){
    echo "$(date) - $1"
}

log "Started script with DEVICE_ID=${DEVICE_ID} K8S_TLS_SECRET_NAME=${K8S_TLS_SECRET_NAME}"

test_tooling_available(){
    if !(command -v $1 &> /dev/null); then
        echo "Required tool \"$1\" is not available. Exiting now."
        exit 100
    fi
}
test_tooling_available "kubectl"
test_tooling_available "openssl"
test_tooling_available "base64"

log "Retrieving certificate from secret $K8S_TLS_SECRET_NAME ..."
SECRET_VALUE=$(kubectl get secret $K8S_TLS_SECRET_NAME -n c8yedge -o jsonpath='{.data.tls\.crt}' | base64 -d)
if [$? -ne 0]; then
    echo "Error while checking TLS secret $K8S_TLS_SECRET_NAME. Exiting now."
    exit 1
fi

COUNT_EXPIRY_THRESHOLD_SECS=5184000
log "Check if certificate is expiring within next $ $COUNT_EXPIRY_THRESHOLD_SECS seconds ..."
echo "${SECRET_VALUE}" | openssl x509 -checkend 5184000 -noout
NEEDS_RENEWAL=$?
if [$NEEDS_RENEWAL -eq 1]; then
    log "Found TLS Cert, no renewal needed. Exiting now."
    exit 0
fi

log "Certificate needs renewal. Requesting a new one..."
./c8y-get-certificate-from-ca renewCert \
    --device-id "$DEVICE_ID" \
    --cumulocity-host $C8Y_HOST \
    --current-certificate "c8y-certificate-$DEVICE_ID.pem" \
    --private-key "c8y-private-key-$DEVICE_ID.pem"

log "Checking renewals exit code ..."
RENEWAL_EXIT_CODE=$?
if [$RENEWAL_EXIT_CODE -eq 0]; then
    log "Received new certificate. Recreating ${K8S_TLS_SECRET_NAME} now ..."
    kubectl delete secret -n c8yedge $K8S_TLS_SECRET_NAME
    kubectl create secret tls $TLS_SECRET_NAME -n c8yedge --cert c8y-certificate-$DEVICE_ID.pem --key c8y-private-key-$DEVICE_ID.pem
    log "Recreated secret"
else
    log "Certificate renewal did not succeed. Exit Code was $RENEWAL_EXIT_CODE. Exiting now"
    exit 1
fi
