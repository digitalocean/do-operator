# `do-operator`: The Kubernetes Operator for DigitalOcean

`do-operator` is a Kubernetes operator for managing and consuming [DigitalOcean](https://www.digitalocean.com/) resources from a Kubernetes cluster.

Currently it supports [DigitalOcean Managed Databases](https://www.digitalocean.com/products/managed-databases).

## Status

**This project is in BETA.**

* This operator should not be depended upon for production use at this time.
* The CRDs in this project are currently `v1alpha1` and may change in the future.
* DigitalOcean supports this project on a best-effort basis via GitHub issues.

**If you have already enabled `do-operator` by clicking `Add database operator` button from cloud control panel UI while creating your `DOKS` cluster, please `DO NOT` install it again.**

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

**Note M1 Macbook users:** this project is made with Kubebuilder which uses `Kustomize v3.8.7` which doesn't have an ARM release and so you need to [manually install kustomize](#manually-installing-kustomize-m1-mac-users) for this step to succeed

2. Build and push your image to the location specified by `IMG` (in the [Makefile](Makefile)):

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

### Manually Installing Kustomize (M1 Mac users)
1. Clone the [Kustomize project](https://github.com/kubernetes-sigs/kustomize)
2. Go to the `v3.5.7` branch: `git checkout kustomize/v3.8.7`
3. Go into the `kustomize` folder (so from the project root it would be `kustomize/kustomize/`)
4. `go build .`
5. Now move the generated `kustomize` binary into the `bin` directory in this project (create it if it doesn't exist `mkdir bin`): `mv kustomize <project path>/bin/kustomize`

### Update Go and dependencies
1. Update Go version in
   1. [go.mod](./go.mod)
   2. [.github/workflows/release.yml](./.github/workflows/release.yml)
   3. [.github/workflows/test.yml](./.github/workflows/test.yml)
   4. [Dockerfile](./Dockerfile)
2. Update Go dependencies
   ```shell
   go get -u ./...
   go mod tidy
   go mod vendor
   ```
3. Run `make test`
4. Create and merge PR

### Release
1. Create release manifest files by running `GITHUB_TOKEN=<redacted> IMG_TAG=vX.Y.Z make release-manifests`
2. Create and merge PR
3. [Trigger the `release` GitHub action workflow](https://github.com/digitalocean/do-operator/actions/workflows/release.yml)
    - Draft a new release [here](https://github.com/digitalocean/do-operator/releases) for the new version
    - Creating a new release will trigger the [release](https://github.com/digitalocean/do-operator/actions/workflows/release.yml) Github Action
    - Follow the [release](https://github.com/digitalocean/do-operator/actions/workflows/release.yml) Github Action until successful completion.

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

