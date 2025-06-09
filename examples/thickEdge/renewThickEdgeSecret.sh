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

cert-file=c8y-certificate-${DEVICE_ID}.pem
priv-key-file=c8y-private-key-${DEVICE_ID}.pem
new-cert-file=c8y-certificate-${DEVICE_ID}.new.pem

log "Certificate needs renewal. Requesting a new one..."
./c8y-certificate-cli renewCert \
    --cumulocity-host $C8Y_HOST \
    --current-certificate "${cert-file}" \
    --private-key "${priv-key-file}" \
    --new-certificate-name "${new-cert-file}"

LAST_EXIT_CODE=$?
if [ $LAST_EXIT_CODE -gt 0 ] ; then
    log "Error while requesting new certificate from ${CLOUD_HOST}. Exit code = ${LAST_EXIT_CODE}."
    log "This is a fatal error. Certificate did not get renewed. Exiting now."
    exit 1
fi

log "Verify certificate..."
./c8y-certificate-cli verifyCert \
  -cumulocity-host "${CLOUD_HOST}" \
  --certificate ${new-cert-file} \
  --private-key ${priv-key-file}

LAST_EXIT_CODE=$?
if [ $LAST_EXIT_CODE -gt 0 ] ; then
    log "Error while verifying certificate against ${CLOUD_HOST}. Exit code = ${LAST_EXIT_CODE}."
    log "This is a fatal error. Certificate did not get renewed. Exiting now."
    exit 1
fi

log "Received new and valid certificate. Recreating ${K8S_TLS_SECRET_NAME} now ..."
kubectl delete secret -n c8yedge $K8S_TLS_SECRET_NAME
log "Deleted secret $K8S_TLS_SECRET_NAME"
kubectl create secret tls $TLS_SECRET_NAME -n c8yedge --cert "${new-cert-file}" --key "${priv-key-file}"
# something went wrong when setting secret
if [$? -gt 0]; then
    log "Error while creating kubernetes secret via kubectl (kubectl reporting exit code > 0)"
    log "This is a fatal error. Certificate did not get renewed. Delete new certificate and exit now."
    rm ${new-cert-file}
    exit 1
fi
# everything succeeded, now swap new and old certificate and delete old one
log "Setting Kubernetes secret succeeded. Swapping old- and new certificate now."
rm ${cert-file}
mv ${new-cert-file} ${cert-file}
log "Certificate renewal succeeded"

exit 0
