# Getting Started with provider-tencentcloud

Install, authenticate, and create your first Tencent Cloud resource with the
provider.

## Install

Install the provider package on your Crossplane cluster:

```bash
kubectl apply -f - <<'YAML'
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-tencentcloud
spec:
  package: xpkg.upbound.io/arindraaribudi/provider-tencentcloud:v1.82.98
YAML
```

Or via the Upbound CLI:

```bash
up ctp provider install xpkg.upbound.io/arindraaribudi/provider-tencentcloud:v1.82.98
```

## Authentication

The provider requires a Tencent Cloud SecretId and SecretKey pair. Create a
Kubernetes Secret and reference it from a `ProviderConfig`:

```bash
kubectl create secret generic tencentcloud-creds \
  --from-literal=secret_id=$TENCENTCLOUD_SECRET_ID \
  --from-literal=secret_key=$TENCENTCLOUD_SECRET_KEY
```

```yaml
apiVersion: pkg.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: tencentcloud
spec:
  credentials:
    source: Secret
    secretRefs:
      - name: tencentcloud-creds
        key: secret_id
      - name: tencentcloud-creds
        key: secret_key
```

## First resource

Create a minimal CVM instance to verify the wiring:

```yaml
apiVersion: cvm.tencentcloud.crossplane.io/v1alpha1
kind: Instance
metadata:
  name: hello-tencentcloud
spec:
  forProvider:
    instanceName: hello
    instanceType: S5.SMALL2
    imageId: img-9qrfxu9d
    availabilityZone: ap-guangzhou-3
    systemDisk:
      diskType: CLOUD_PREMIUM
      diskSize: 20
  providerConfigRef:
    name: tencentcloud
```

Apply, then watch:

```bash
kubectl get managed
```

## Next steps

See the `examples-generated/` directory for full coverage of every supported
managed resource, including VPCs, CBS volumes, RDS databases, TKE clusters, and
CLB load balancers.