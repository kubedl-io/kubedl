# Metadata Persistency

## Background

As AI platform builders, it is quite inefficient to directly query metadata(jobs/pods) from ApiServer for following reasons:

- intensive queries may cause ApiServer jitters and does negative impact on cluster reliability.
- metadata will be permanently lost once job was deleted from etcd.
- etcd will possibly be the bottleneck of whole system.

To address this, `KubeDL` provides pluggable facilities for developers/users to persist their metadata to customized external storage system, 
such as `mysql`, `postgresql`, `redis`... and `mysql` storage plugin has now available.

## How To Use

To enable metadata persistency, users should set-up their storage system certificates first, hence `KubeDL` will be able to connect to.
Take `mysql` as a guideline, we can enable metadata persistency by following steps:

1. apply certificates leveraging `Service` or `ConfigMap`, `Service` is strongly recommended for security.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kubedlpro-mysql-config
  namespace: kubedl-system
type: Opaque
stringData:
  host: my.host.com
  dbName: kubedl
  user: kubedl-user
  password: this-is-me
  port: "3306"
```

2. referent certificates stored in `Secret` by amending `KubeDL` deployment, then main process can load certificates by environments.
3. at this point, we just need to add a startup flag in `KubeDL` deployment to declare which storage plugin to be activated, a rendered yaml is down below: 

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubedl
  namespace: kubedl-system
  labels:
    app: kubedl
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kubedl
  template:
    metadata:
      labels:
        app: kubedl
    spec:
      containers:
      - image: kubedl/kubedl:v0.3.0
        imagePullPolicy: Always
        name: kubedl-manager
        args:
        - "--object-storage"
        - "mysql"
        env:
        - name: MYSQL_HOST
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-mysql-config
              key: host
        - name: MYSQL_DB_NAME
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-mysql-config
              key: dbName
        - name: MYSQL_USER
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-mysql-config
              key: user
        - name: MYSQL_PASSWORD
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-mysql-config
              key: password
```

now a persistent controller driven by `mysql` has been enabled and you may check the data meets expectations or not in your storage service.

## Contributions

`KubeDL` now supports `mysql` storage plugin only, if you have needs on other storage protocols like `mangodb`, `influxdb`, 
contributions are welcomed, and developers just need to implement a storage plugin interface :)