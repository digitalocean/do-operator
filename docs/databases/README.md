# Databases

`do-operator` supports managing and connecting to [DigitalOcean Databases](https://www.digitalocean.com/products/managed-databases).
The following functionality is supported:
* Managing a database cluster via the `DatabaseCluster` CRD.
* Connecting to an existing database via the `DatabaseClusterReference` CRD.
* Managing a database user via the `DatabaseUser` CRD.
* Getting credentials for an existing user via the `DatabaseUserReference` CRD.

The details of these CRDs are described below.

## The `DatabaseCluster` CRD

The `DatabaseCluster` CRD is used to create and manage the lifecycle of a DigitalOcean Database cluster.
Note that when using this CRD, the database's lifecycle and configuration will be fully managed by the operator, and you should not make changes to its configuration directly (e.g., do not resize it through the DigitalOcean control panel).
When you delete a `DatabaseCluster` object the associated database *will* be deleted by the operator.

The `DatabaseCluster` CRD looks like this:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-app-db
  namespace: my-application
spec:
  engine: mysql
  name: my-app-db
  version: '8'
  numNodes: 1
  size: db-s-1vcpu-1gb
  region: tor1
```

The [DigitalOcean API reference](https://docs.digitalocean.com/reference/api/api-reference/#tag/Databases) lists the valid options for each field of the spec.
All fields are required, and the configuration will be validated by a validating webhook when a `DatabaseCluster` manifest is applied.

Once the operator has created the database, the status will be filled in with details about the database:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseCluster
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"databases.digitalocean.com/v1alpha1","kind":"DatabaseCluster","metadata":{"annotations":{},"name":"my-app-db","namespace":"my-application"},"spec":{"engine":"mysql","name":"my-app-db","numNodes":1,"region":"tor1","size":"db-s-1vcpu-1gb","version":"8"}}
  creationTimestamp: "2022-09-14T22:30:10Z"
  finalizers:
  - databases.digitalocean.com
  generation: 1
  name: my-app-db
  namespace: my-application
  resourceVersion: "19905"
  uid: d3f021a2-e732-45e6-97d2-9e07577dcaea
spec:
  engine: mysql
  name: my-app-db
  version: '8'
  numNodes: 1
  size: db-s-1vcpu-1gb
  region: tor1
status:
  createdAt: "2022-09-14T22:31:51Z"
  status: creating
  uuid: 74d2d156-8cb2-4732-92c5-3150cde33a10
```

Additionally, two `ConfigMap` objects will be created containing public and private connection information for the database:

```yaml
---
apiVersion: v1
data:
  database: defaultdb
  host: my-app-db-do-user-xxx-0.b.db.ondigitalocean.com
  port: "25060"
  ssl: "true"
kind: ConfigMap
metadata:
  creationTimestamp: "2022-09-14T22:31:51Z"
  name: my-app-db-connection
  namespace: my-application
  ownerReferences:
  - apiVersion: databases.digitalocean.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: DatabaseCluster
    name: my-app-db
    uid: d3f021a2-e732-45e6-97d2-9e07577dcaea
  resourceVersion: "22956"
  uid: f2082857-4bfb-4618-b07a-e8cec16475ca
---
apiVersion: v1
data:
  database: defaultdb
  host: private-my-app-db-do-user-xxx-0.b.db.ondigitalocean.com
  port: "25060"
  ssl: "true"
kind: ConfigMap
metadata:
  creationTimestamp: "2022-09-14T22:31:51Z"
  name: my-app-db-private-connection
  namespace: my-application
  ownerReferences:
  - apiVersion: databases.digitalocean.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: DatabaseCluster
    name: my-app-db
    uid: d3f021a2-e732-45e6-97d2-9e07577dcaea
  resourceVersion: "23235"
  uid: 336aeb22-bd75-4e3c-93b4-e8b5755127c9
```

A `Secret` will be created containing the default credentials (except for MongoDB, where default credentials cannot be retrieved):

```yaml
apiVersion: v1
data:
  password: cGFzc3dvcmQK
  private_uri: bXlzcWw6Ly9kb2FkbWluOnBhc3N3b3JkQHByaXZhdGUtbXktYXBwLWRiLWRvLXVzZXIteHh4LTAuYi5kYi5vbmRpZ2l0YWxvY2Vhbi5jb206MjUwNjAvZGVmYXVsdGRiP3NzbC1tb2RlPVJFUVVJUkVECg==
  uri: bXlzcWw6Ly9kb2FkbWluOnBhc3N3b3JkQG15LWFwcC1kYi1kby11c2VyLXh4eC0wLmIuZGIub25kaWdpdGFsb2NlYW4uY29tOjI1MDYwL2RlZmF1bHRkYj9zc2wtbW9kZT1SRVFVSVJFRAo=
  username: ZG9hZG1pbg==
kind: Secret
metadata:
  creationTimestamp: "2022-09-14T22:31:51Z"
  name: my-app-db-default-credentials
  namespace: my-application
  ownerReferences:
  - apiVersion: databases.digitalocean.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: DatabaseCluster
    name: my-app-db
    uid: d3f021a2-e732-45e6-97d2-9e07577dcaea
  resourceVersion: "23518"
  uid: ac6f81df-36eb-4b7f-8640-2287ea88fd1d
type: Opaque
```

## The `DatabaseClusterReference` CRD

The `DatabaseClusterReference` CRD is used to simplify connecting to an existing DigitalOcean Database cluster from your Kubernetes cluster.
When using this CRD, the database must already exist and you can manage it using any tool you prefer (e.g., manually in the DigitalOcean control panel or with Terraform).
When you delete a `DatabaseClusterReference` object the associated database *will not* be deleted by the operator.

The `DatabaseClusterReference` CRD looks like this:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseClusterReference
metadata:
  name: my-app-db
  namespace: my-application
spec:
  uuid: 74d2d156-8cb2-4732-92c5-3150cde33a10
```

You can find the UUID of your database with the [doctl](https://github.com/digitalocean/doctl) CLI or in the DigitalOcean control panel.

Once the operator has processed the `DatabaseClusterReference`, the status will be filled in with details about the database:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseClusterReference
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"databases.digitalocean.com/v1alpha1","kind":"DatabaseClusterReference","metadata":{"annotations":{},"name":"my-app-db","namespace":"my-application"},"spec":{"uuid":"74d2d156-8cb2-4732-92c5-3150cde33a10"}}
  creationTimestamp: "2022-09-14T22:34:13Z"
  generation: 1
  name: my-app-db
  namespace: my-application
  resourceVersion: "20563"
  uid: a29c30c9-79fe-435c-ab0e-eb2db2809007
spec:
  uuid: 74d2d156-8cb2-4732-92c5-3150cde33a10
status:
  createdAt: "2022-09-14T22:31:51Z"
  engine: mysql
  name: my-app-db
  numNodes: 1
  region: tor1
  size: db-s-1vcpu-1gb
  status: creating
  version: "8"
```

Additionally, two `ConfigMap` objects will be created containing public and private connection information for the database:

```yaml
---
apiVersion: v1
data:
  database: defaultdb
  host: my-app-db-do-user-xxx-0.b.db.ondigitalocean.com
  port: "25060"
  ssl: "true"
kind: ConfigMap
metadata:
  creationTimestamp: "2022-09-14T22:31:51Z"
  name: my-app-db-connection
  namespace: my-application
  ownerReferences:
  - apiVersion: databases.digitalocean.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: DatabaseClusterReference
    name: my-app-db
    uid: d3f021a2-e732-45e6-97d2-9e07577dcaea
  resourceVersion: "22956"
  uid: f2082857-4bfb-4618-b07a-e8cec16475ca
---
apiVersion: v1
data:
  database: defaultdb
  host: private-my-app-db-do-user-xxx-0.b.db.ondigitalocean.com
  port: "25060"
  ssl: "true"
kind: ConfigMap
metadata:
  creationTimestamp: "2022-09-14T22:31:51Z"
  name: my-app-db-private-connection
  namespace: my-application
  ownerReferences:
  - apiVersion: databases.digitalocean.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: DatabaseClusterReference
    name: my-app-db
    uid: d3f021a2-e732-45e6-97d2-9e07577dcaea
  resourceVersion: "23235"
  uid: 336aeb22-bd75-4e3c-93b4-e8b5755127c9
```

A `Secret` will be created containing the default credentials (except for MongoDB, where default credentials cannot be retrieved):

```yaml
apiVersion: v1
data:
  password: cGFzc3dvcmQK
  private_uri: bXlzcWw6Ly9kb2FkbWluOnBhc3N3b3JkQHByaXZhdGUtbXktYXBwLWRiLWRvLXVzZXIteHh4LTAuYi5kYi5vbmRpZ2l0YWxvY2Vhbi5jb206MjUwNjAvZGVmYXVsdGRiP3NzbC1tb2RlPVJFUVVJUkVECg==
  uri: bXlzcWw6Ly9kb2FkbWluOnBhc3N3b3JkQG15LWFwcC1kYi1kby11c2VyLXh4eC0wLmIuZGIub25kaWdpdGFsb2NlYW4uY29tOjI1MDYwL2RlZmF1bHRkYj9zc2wtbW9kZT1SRVFVSVJFRAo=
  username: ZG9hZG1pbg==
kind: Secret
metadata:
  creationTimestamp: "2022-09-14T22:31:51Z"
  name: my-app-db-default-credentials
  namespace: my-application
  ownerReferences:
  - apiVersion: databases.digitalocean.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: DatabaseClusterReference
    name: my-app-db
    uid: d3f021a2-e732-45e6-97d2-9e07577dcaea
  resourceVersion: "23518"
  uid: ac6f81df-36eb-4b7f-8640-2287ea88fd1d
type: Opaque
```

## The `DatabaseUser` CRD

The `DatabaseUser` CRD is used to create and manage the lifecycle of a user in a DigitalOcean Database cluster.
Note that when using this CRD, the user's lifecycle and configuration will be fully managed by the operator, and you should not make changes to it directly (e.g., do not delete the user through the DigitalOcean control panel).
When you delete a `DatabaseUser` object the associated user *will* be deleted by the operator and any applications using it will lose access to your database.

The `DatabaseUser` references either a `DatabaseCluster` or a `DatabaseClusterReference`.
You must create one of those resources before you can create a `DatabaseUser`.

The `DatabaseUser` CRD looks like this:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseUser
metadata:
  name: my-app-db-user
  namespace: my-application
spec:
  databaseCluster:
    apiGroup: databases.digitalocean.com
    kind: DatabaseCluster
    name: my-app-db
  username: my_app_user
```

The `databaseCluster` field can also refer to a `DatabaseClusterReference`.
See the [best practices](#BestPractices) below for suggestions on when you may want to use each option.

Once the operator has created the user, the status will be filled in with details about the user:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseUser
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"databases.digitalocean.com/v1alpha1","kind":"DatabaseUser","metadata":{"annotations":{},"name":"databasecluster-user","namespace":"default"},"spec":{"databaseCluster":{"apiGroup":"databases.digitalocean.com","kind":"DatabaseCluster","name":"my-app-db"},"username":"my_app_user"}}
  creationTimestamp: "2022-09-14T22:50:30Z"
  finalizers:
  - databases.digitalocean.com
  generation: 1
  name: my-app-db-user
  namespace: my-application
  resourceVersion: "25094"
  uid: 8ab3a34c-aafd-4bbd-a38a-3b7aa0a51218
spec:
  databaseCluster:
    apiGroup: databases.digitalocean.com
    kind: DatabaseCluster
    name: my-app-db
  username: my_app_user
status:
  clusterUUID: 74d2d156-8cb2-4732-92c5-3150cde33a10
  role: normal
```

Additionally, a `Secret` will be created containing the user's credentials:

```yaml
apiVersion: v1
data:
  password: cGFzc3dvcmQK
  username: c2FtcGxlX3VzZXJfMQ==
kind: Secret
metadata:
  creationTimestamp: "2022-09-14T22:50:32Z"
  name: my-app-db-user-credentials
  namespace: my-application
  ownerReferences:
  - apiVersion: databases.digitalocean.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: DatabaseUser
    name: my-app-db-user
    uid: 8ab3a34c-aafd-4bbd-a38a-3b7aa0a51218
  resourceVersion: "25090"
  uid: da571e9f-4b5a-448d-a010-829fc9c925fe
type: Opaque
```

## The `DatabaseUserReference` CRD

The `DatabaseUserReference` CRD is used to simplify connecting to a DigitalOcean Database cluster with an exsiting database user.
When using this CRD, the user must already exist and you can manage it using any tool you prefer (e.g., manually in the DigitalOcean control panel or with Terraform).
When you delete a `DatabaseUserReference` the associated user *will not* be deleted by the operator.

The `DatabaseUserReference` references either a `DatabaseCluster` or a `DatabaseClusterReference`.
You must create one of those resources before you can create a `DatabaseUserReference`.

The `DatabaseUserReference` CRD looks like this:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseUserReference
metadata:
  name: my-app-db-user-ref
  namespace: my-application
spec:
  databaseCluster:
    apiGroup: databases.digitalocean.com
    kind: DatabaseCluster
    name: my-app-db
  username: my_user
```

The `databaseCluster` field can also refer to a `DatabaseClusterReference`.
See the [best practices](#BestPractices) below for suggestions on when you may want to use each option.

Once the operator has processed the `DatabaseUserReference`, the status will be filled in with details about the user:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseUserReference
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"databases.digitalocean.com/v1alpha1","kind":"DatabaseUserReference","metadata":{"annotations":{},"name":"my-app-db-user-ref","namespace":"my-application"},"spec":{"databaseCluster":{"apiGroup":"databases.digitalocean.com","kind":"DatabaseCluster","name":"my-app-db"},"username":"my_user"}}
  creationTimestamp: "2022-09-14T22:54:04Z"
  generation: 1
  name: my-app-db-user-ref
  namespace: my-application
  resourceVersion: "26079"
  uid: f5218c9c-ab15-4024-8ac7-bf60e421ce0e
spec:
  databaseCluster:
    apiGroup: databases.digitalocean.com
    kind: DatabaseCluster
    name: my-app-db
  username: my_user
status:
  clusterUUID: 74d2d156-8cb2-4732-92c5-3150cde33a10
  role: normal
```

Additionally, a `Secret` will be created containing the user's credentials:

```yaml
apiVersion: v1
data:
  password: cGFzc3dvcmQK
  username: bXlfdXNlcg==
kind: Secret
metadata:
  creationTimestamp: "2022-09-14T22:54:04Z"
  name: my-app-db-user-ref-credentials
  namespace: my-application
  ownerReferences:
  - apiVersion: databases.digitalocean.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: DatabaseUserReference
    name: my-app-db-user-ref
    uid: f5218c9c-ab15-4024-8ac7-bf60e421ce0e
  resourceVersion: "26081"
  uid: 3d2e9a64-5229-44cb-ba33-95086bcb1892
type: Opaque
```

Note that since DigitalOcean MongoDB databases do not support retrieving credentials for existing users, you cannot create a `DatabaseUserReference` for a MongoDB database.

## Best Practices

We suggest using one of the two architectures below to manage databases and database users with this operator.

### Referenced Database Architecture

In the referenced database architecture, the database is managed separately (e.g., using Terraform) and the operator is used to manage and simplify application access to it.
Database users in this architecture are managed by the operator, since it is desirable to use a separate database user for each application.
This is the preferred architecture for most use-cases, where the database may be used by applications running in more than one Kubernetes cluster, and the database should continue to exist after the Kubernetes cluster is deleted.

In this architecture, you may choose to create and manage your database using Terraform:

```hcl
resource "digitalocean_database_cluster" "my-app-db" {
  name       = "my-app-db"
  engine     = "mysql"
  version    = "8"
  size       = "db-s-1vcpu-1gb"
  region     = "tor1"
  node_count = 1
}
```

You will then reference that database using a `DatabaseClusterReference` object:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseClusterReference
metadata:
  name: my-app-db
  namespace: my-application
spec:
  uuid: 28e014ef-c0cc-4610-92af-14f3acc35bce
```

And you will create a user in the database with a `DatabaseUser` object:

```yaml
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseUser
metadata:
  name: my-app-db-user
  namespace: my-application
spec:
  databaseCluster:
    apiGroup: databases.digitalocean.com
    kind: DatabaseClusterReference
    name: my-app-db
  username: my_app_user
```

You can then reference the connection information and user credentials in your workload manifests.
For example, when configuring an application to connect to your MySQL database, you might configure the following environment variables for the container:

```yaml
          env:
            - name: MYSQL_HOST
              valueFrom:
                configMapKeyRef:
                  name: guestbook-db-private-connection
                  key: host
            - name: MYSQL_PORT
              valueFrom:
                configMapKeyRef:
                  name: guestbook-db-private-connection
                  key: port
            - name: MYSQL_DATABASE
              valueFrom:
                configMapKeyRef:
                  name: guestbook-db-private-connection
                  key: database
            - name: MYSQL_USER
              valueFrom:
                secretKeyRef:
                  name: guestbook-db-user-credentials
                  key: username
            - name: MYSQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: guestbook-db-user-credentials
                  key: password
```

### Managed Database Architecture

In the managed database architecture, the operator manages the lifecycle and configuration of the database.
This is appropriate when your database will only ever be used by applications running in a single Kubernetes cluster, and the database's lifetime should match the Kubernetes cluster's lifetime.
For example, this architecture is a good choice when using Redis as a cache for a specific application.

In this architecture, you will create a `DatabaseCluster` object and, if relevant for your chosen database engine, a `DatabaseUser` object:

```yaml
---
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseCluster
metadata:
  name: my-app-db
  namespace: my-application
spec:
  engine: mysql
  name: my-app-db
  version: '8'
  numNodes: 1
  size: db-s-1vcpu-1gb
  region: tor1

---
apiVersion: databases.digitalocean.com/v1alpha1
kind: DatabaseUser
metadata:
  name: my-app-db-user
  namespace: my-application
spec:
  databaseCluster:
    apiGroup: databases.digitalocean.com
    kind: DatabaseCluster
    name: my-app-db
  username: my_app_user
```

You can then reference the connection information and user credentials in your workload manifests.
For example, when configuring an application to connect to your MySQL database, you might configure the following environment variables for the container:

```yaml
          env:
            - name: MYSQL_HOST
              valueFrom:
                configMapKeyRef:
                  name: guestbook-db-private-connection
                  key: host
            - name: MYSQL_PORT
              valueFrom:
                configMapKeyRef:
                  name: guestbook-db-private-connection
                  key: port
            - name: MYSQL_DATABASE
              valueFrom:
                configMapKeyRef:
                  name: guestbook-db-private-connection
                  key: database
            - name: MYSQL_USER
              valueFrom:
                secretKeyRef:
                  name: guestbook-db-user-credentials
                  key: username
            - name: MYSQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: guestbook-db-user-credentials
                  key: password
```

## Limitations

* The operator does not currently support upgrading database cluster versions.
* The operator does not currently support migrating database clusters between regions.
* The operator does not currently support any engine-specific configuration.
* It is not currently possible to create a `DatabaseCluster` to manage an existing database.
* All databases will be created in your account's default VPC in the chosen region.
* All DigitalOcean Databases product limitations apply.
  Notably:
  * Redis databases do not support user management.
    You cannot create `DatabaseUser` or `DatabaseUserReference` resources that reference a Redis database.
  * You cannot retrieve the password for a MongoDB user after it has been created.
    Therefore, you cannot create `DatabaseUserReference` resources for a MongoDB database.
