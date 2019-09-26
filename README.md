# DO DB K8S operator

```shell
operator-sdk new do-operator --repo github.com/digitalocean/do-operator
```

```shell
operator-sdk add api --api-version=doop.do.co/v1alpha1 --kind=Database
operator-sdk add controller --api-version=doop.do.co/v1alpha1 --kind=Database
```
