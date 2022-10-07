# `do-operator`: The Kubernetes Operator for DigitalOcean

`do-operator` is a Kubernetes operator for managing and consuming [DigitalOcean](https://www.digitalocean.com/) resources from a Kubernetes cluster.

Currently it supports [DigitalOcean Managed Databases](https://www.digitalocean.com/products/managed-databases).

## Status

**This project is in BETA.**

* This operator should not be depended upon for production use at this time.
* The CRDs in this project are currently `v1alpha1` and may change in the future.
* DigitalOcean supports this project on a best-effort basis via GitHub issues.

## Quick Start

To install the operator on a Kubernetes cluster, you can follow these steps:

1. Install cert-manager (if not already installed):
```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
```
2. Generate a [DigitalOcean API token](https://docs.digitalocean.com/reference/api/create-personal-access-token/) and base64 encode it.
3. Edit the manifest for the most recent version in the `releases/` directory. Find the `do-operator-do-api-token` Secret and replace the `access-token` value with your base64-encoded token:
```yaml
apiVersion: v1
data:
  access-token: <your base64-encoded token goes here>
kind: Secret
metadata:
  name: do-operator-do-api-token
  namespace: do-operator-system
type: Opaque
```
4. Deploy the manifest:
```sh
kubectl apply -f releases/do-operator-<version>.yaml
```

## Usage

See the full documentation in the [docs](docs/) directory.

## Development

The `do-operator` is built using [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).
The [Kubebuilder Book](https://book.kubebuilder.io/) is a useful reference for understanding how the pieces fit together.

To test your changes you will need a Kubernetes cluster to run against.
We suggest using [DigitalOcean Kubernetes](https://docs.digitalocean.com/products/kubernetes/), but you may use [KIND](https://sigs.k8s.io/kind) or any other cluster for testing.
The following will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

**Note that you will be billed for any DigitalOcean resources you create via the operator while testing.**
Make sure you are aware of [DigitalOcean's pricing](https://www.digitalocean.com/pricing) and clean up resources when you're finished testing.

### Building and Installing

1. Install the Custom Resource Definitions:

```sh
make install
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/do-operator:tag
```
	
3. [Deploy cert-manager](https://cert-manager.io/docs/installation/), which is necessary to manage certificates for the webhooks.

4. Generate a [DigitalOcean API token](https://docs.digitalocean.com/reference/api/create-personal-access-token/) to use for testing.

5. Create a local environment file containing your API token. Note that this file is in the `.gitignore` so it will remain local to your machine:

```sh
cat <<EOF > config/manager/do-api-token.env
access-token=<your api token here>
EOF
```

The contents of this file will be used to create a secret in the cluster, which is used by the operator deployment to manage resources in your DigitalOcean account.

6. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/do-operator:tag
```

### Undeploying the Controller

To undeploy the controller from the cluster:

```sh
make undeploy
```

### Uninstalling the CRDs

To delete the CRDs from the cluster:

```sh
make uninstall
```

## Contributing

At DigitalOcean we value and love our community!
If you have any issues or would like to contribute, see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Copyright 2022 DigitalOcean.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

