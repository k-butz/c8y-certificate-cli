#!/bin/sh

# Configuration
DEVICE_ID=kb-dev-01 # this is used as name for the cloud-device
CLOUD_HOST=https://iot.cumulocity.com
CLOUD_TENANT_ID=t1234
CLOUD_USER=korbinian.butz@cumulocity.com
CLOUD_PASSWORD="..."
K8S_TLS_SECRET_NAME="c8y-cloud-tls-secret"

log(){
    echo "$(date) - $1"
}

log "Started script with DEVICE_ID=${DEVICE_ID} CLOUD_HOST=${CLOUD_HOST} CLOUD_TENANT_ID=${CLOUD_TENANT_ID} CLOUD_USER=${CLOUD_USER} K8S_TLS_SECRET_NAME=${K8S_TLS_SECRET_NAME}"

# Test if required toolings are available. Exit when not.
test_tooling_available(){
    if !(command -v $1 &> /dev/null); then
        log "Required tool \"$1\" is not available. Exiting now."
        exit 100
    fi
}
test_tooling_available "kubectl"

# 1. Create CSR, register device, retrieve certificate
# Adapt executable to the one that fits your OS and cpu (e.g. to use ./c8y-certificate-cli_linux_amd64 instead)
log "Retrieving certificates from Cloud ..."
log "Certificate retrieval logs:"
echo "====================================================================="
./c8y-certificate-cli_darwin_arm64 \
    --device-id "${DEVICE_ID}" \
    --cumulocity-host "${CLOUD_HOST}" \
    --cumulocity-tenant-id "${CLOUD_TENANT_ID}" \
    --cumulocity-user "${CLOUD_USER}" \
    --cumulocity-password "${CLOUD_PASSWORD}"
echo "====================================================================="

cert-file=c8y-certificate-${DEVICE_ID}.pem
priv-key-file=c8y-private-key-${DEVICE_ID}.pem

LAST_EXIT_CODE=$?
if [ $LAST_EXIT_CODE -gt 0 ] ; then
    log "Error while retrieving certificate from ${CLOUD_HOST}. Exit code = ${LAST_EXIT_CODE}. For details, have a look at the logs from executable."
    log "This is a fatal error. Exiting now."
    exit 1
fi

log "Verify certificate"
# 2. Just ot be sure, test if certificate is valid 
./c8y-certificate-cli verifyCert \
  -cumulocity-host "${CLOUD_HOST}" \
  --certificate ${cert-file} \
  --private-key ${priv-key-file}

LAST_EXIT_CODE=$?
if [ $LAST_EXIT_CODE -gt 0 ] ; then
    log "Error while verifying certificate against ${CLOUD_HOST}. Exit code = ${LAST_EXIT_CODE}."
    log "This is a fatal error. Exiting now."
    exit 1
fi

# 3. Create kubernetes secret
log "Registering .pem files from previous step as Kubernetes secret now ..." 
kubectl create secret tls ${K8S_TLS_SECRET_NAME} -n c8yedge \
    --cert="c8y-certificate-${DEVICE_ID}.pem" \
    --key="c8y-private-key-${DEVICE_ID}.pem"
log "Created Kubernetes secret '${K8S_TLS_SECRET_NAME}'"

# 3. Merge the secret from step 2 into c8y-edge.yaml
# 
# For below instruction the tooling 'yq' is required (https://github.com/mikefarah/yq), it is a single binary available for different OS and CPU
# Uncomment the proper line that applies to your environment
# wget -q https://github.com/mikefarah/yq/releases/latest/download/yq_linux_arm64 -O yq
# wget -q https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O yq
# wget -q https://github.com/mikefarah/yq/releases/latest/download/yq_darwin_arm64 -O yq
# wget -q https://github.com/mikefarah/yq/releases/latest/download/yq_darwin_amd64 -O yq
# wget -q https://github.com/mikefarah/yq/releases/latest/download/yq_windows_arm64 -O yq
# wget -q https://github.com/mikefarah/yq/releases/latest/download/yq_windows_amd64 -O yq
# 

chmod +x ./yq

# Send cloud tenant configuration (including secret reference) to a file
FILE_CONTENT="$(cat <<-EOF
spec:
  cloudTenant: 
    domain: "${CLOUD_HOST}"
    tlsSecretName: "${K8S_TLS_SECRET_NAME}"
EOF
)"
YML_CFG_FILE_NAME="cloud-tenant-configuration.yml"
log "${FILE_CONTENT}" > "${YML_CFG_FILE_NAME}"

# Merge cloud-tenant-configuration into c8yedge.yaml file
log "Merge cloud-tenant configuration in c8yedge.yaml now ..."
./yq ". *= load(\"${YML_CFG_FILE_NAME}\")" c8yedge.yaml > c8yedge.with-cloud-secret.yaml
log "Produced file 'c8yedge.with-cloud-secret.yaml'"

# Cleanup
log "Cleaning up..."
rm "./${YML_CFG_FILE_NAME}" 2> /dev/null
log "Cleanup done. Exiting now."

# Optionally, overwrite existing c8yedge yaml
# Might be a good idea to back up your original c8yedge.yaml before deleting
# rm c8yedge.yaml
# mv c8yedge.with-cloud-secret.yaml c8yedge.yaml

# Now do kubectl apply for the changed in c8yedge.yaml to take effect