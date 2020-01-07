A Kubernetes [operator](https://github.com/operator-framework/operator-sdk) for provisioning and managing [DigitalOcean Databases](https://www.digitalocean.com/products/managed-databases/) as Custom Resources in your cluster.

[![CircleCI](https://circleci.com/gh/digitalocean/do-operator.svg?style=svg)](https://circleci.com/gh/digitalocean/do-operator)

## Usage

Create a `Secret` containing your DigitalOcean API token:
```sh
kubectl create secret generic do-operator --from-literal="DIGITALOCEAN_ACCESS_TOKEN=${DIGITALOCEAN_ACCESS_TOKEN}"
```

Install the latest release of `do-operator` into your cluster.
```sh
kubectl apply -f https://raw.githubusercontent.com/digitalocean/do-operator/master/releases/v0.0.3/manifest.yaml
```

Create a `Database` object and wait + watch as the operator creates and monitors the status of a DO database with the given configuration.
```yaml
apiVersion: doop.do.co/v1alpha1
kind: Database
metadata:
  name: example
spec:
  name: example
  engine: redis
  version: "5"
  size: db-s-2vcpu-4gb
  region: sfo2
  numNodes: 2
  tags:
  - test
  - doop
```

Check the status:
```sh
kubectl describe database example
```

A couple secrets will be created for each `Database` instance; `example-connection` and `example-private-connection`.

Use them in your workload:

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: hello
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: hello
        image: snormore/hello
        env:
          - name: DATABASE_URI
            valueFrom:
              secretKeyRef:
                name: example-connection
                key: uri
```
