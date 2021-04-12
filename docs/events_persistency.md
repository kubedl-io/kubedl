# Events Persistency

## Background

Similar to `metadata persistency`, platform builders may also wants to persist job-related events to external system(usually time-series databases), 
since events can only be preserved for 3 hours by default in kubernetes, and events sinks helps a lot in debugging and tracing.
`KubeDL` also provides pluggable facilities to persist events to customized external storage system, such as `aliyun-sls`, `graphite`...and `aliyun-sls` plugin has now available.

## How To Use

Same as `metadata persistency`, users should set-up their storage system certificates first.

1. apply certificates leveraging `Service` or `ConfigMap`, `Service` is strongly recommended for security.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kubedl-sls-config
  namespace: kubedl-system
type: Opaque
stringData:
  endpoint: zhangbei.log.aliyuncs.com
  accessKey: my-ak
  accessSecret: my-sk
  project: kubedl-project
  logStore: kubedl
```

2. referent certificates stored in `Secret` by amending `KubeDL` deployment, then main process can load certificates by environments.
3. at this point, we just need to add a startup flag in `KubeDL` deployment to declare which events plugin to be activated, a rendered yaml is down below:

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
        - "--event-storage"
        - "aliyun-sls"
        env:
        - name: SLS_ENDPOINT
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-sls-config
              key: endpoint
        - name: SLS_KEY_ID
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-sls-config
              key: accessKey
        - name: SLS_KEY_SECRET
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-sls-config
              key: accessSecret
        - name: SLS_PROJECT
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-sls-config
              key: project
        - name: SLS_LOG_STORE
          value:
          valueFrom:
            secretKeyRef:
              name: kubedl-sls-config
              key: logStore
```

## Contributions

`KubeDL` now supports `aliyun-sls` events plugin only, if you have needs on other storage protocols,
contributions are welcomed, and developers just need to implement an event storage plugin interface :)