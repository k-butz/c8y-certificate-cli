**About**

A collection of commands that have been useful for testing/debugging purposes.

**Certificate lifecycle management**

```sh
DEVICE_ID=kb_edge_ab128
./c8y-certificate-cli registerUsingPassword \
  --device-id "$DEVICE_ID" \
  --cumulocity-host $C8Y_HOST \
  --cumulocity-tenant-id $C8Y_TENANT \
  --cumulocity-user 'korbinian.butz@cumulocity.com' \
  --cumulocity-password "$C8Y_PASSWORD"

sudo cp "c8y-private-key-$DEVICE_ID.pem" $(tedge config get 'device.key_path')
sudo cp "c8y-certificate-$DEVICE_ID.pem" $(tedge config get 'device.cert_path')
sudo tedge connect c8y

# renew
./c8y-certificate-cli renewCert \
  --cumulocity-host $C8Y_HOST \
  --current-certificate "c8y-certificate-$DEVICE_ID.pem" \
  --private-key "c8y-private-key-$DEVICE_ID.pem"

sudo cp "c8y-certificate-$DEVICE_ID.new.pem" $(tedge config get 'device.cert_path')
sudo tedge reconnect c8y
```

**thin-edge.io snippets**

```sh
tedge mqtt pub 'te/device/main///e/login_event' '{
  "text": "A user just logged in"
}'
```

**mosquitto snippets**

```sh
mosquitto_pub \
    --key ./c8y-private-key-kb_ux_25195.pem \
    --cert ./c8y-certificate-kb_ux_25195.pem \
    -h kb.latest.stage.c8y.io -t s/us -p 8883 \
    -i kb_ux_25195  \
    -m '400,c8y_MyEvent,"Something was triggered"' \
    --cafile /etc/ssl/certs/ca-certificates.crt \
    --debug
```

**Edge K8S**

```sh
DEVICE_ID=kb-test-01-cd
TLS_SECRET_NAME=kb-stg-tls-secret

kubectl get pods -n c8yedge | grep -i thin

kubectl get secret -n c8yedge $TLS_SECRET_NAME
kubectl describe secret -n c8yedge $TLS_SECRET_NAME
kubectl delete secret -n c8yedge $TLS_SECRET_NAME
kubectl create secret tls $TLS_SECRET_NAME -n c8yedge --cert c8y-certificate-$DEVICE_ID.pem --key c8y-private-key-$DEVICE_ID.pem

kubectl edit edge -n c8yedge
```
