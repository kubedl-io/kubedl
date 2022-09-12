# Run a Torch Elastic Job with KubeDL Operator

This tutorial walks you through an example to run a Torch Elastic Job. [Torch Elastic](https://pytorch.org/elastic/0.2.0/) enables distributed PyTorch training jobs to be executed in a fault-tolerant and elastic manner.
Torch Elastic can be used in the following cases:
- Fault Tolerance: jobs that run on infrastructure where nodes get replaced frequently, either due to flaky hardware or by design. Or mission critical production grade jobs that need to be run with resilience to failures.
- Dynamic Capacity Management: jobs that run on leased capacity that can be taken away at any time (e.g. AWS spot instances) or shared pools where the pool size can change dynamically based on demand.

## Requirements

#### 1. Deploy KubeDL
Follow the [installation tutorial](https://github.com/alibaba/kubedl#getting-started) in README and deploy `kubedl` operator to cluster.

#### 2. Apply Pytorch Job CRD

`PytorchJob` CRD(CustomResourceDefinition) manifest file describes the structure of a pytorch job spec. Run the following to apply the CRD:

```bash
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/config/crd/bases/training.kubedl.io_pytorchjobs.yaml
``` 

## Run Torch Elastic Job on Kubernetes with KubeDL

Run `torch elastic` job on kubernetes natively.
### 1. Deploy an etcd server.
Create a new namespace `elastic-job`.
```bash
kubectl create ns elastic-job
```
Create an etcd server and a service `etcd-service` with port 2379.
```yaml
apiVersion: v1
kind: Service
metadata:
  name: etcd-service
  namespace: elastic-job
spec:
  ports:
  - name: etcd-client-port
    port: 2379
    protocol: TCP
    targetPort: 2379
  selector:
    app: etcd

---
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: etcd
  name: etcd
  namespace: elastic-job
spec:
  containers:
  - command:
    - /usr/local/bin/etcd
    - --data-dir
    - /var/lib/etcd
    - --enable-v2
    - --listen-client-urls
    - http://0.0.0.0:2379
    - --advertise-client-urls
    - http://0.0.0.0:2379
    - --initial-cluster-state
    - new
    image: k8s.gcr.io/etcd:3.5.1-0
    name: etcd
    ports:
    - containerPort: 2379
      name: client
      protocol: TCP
    - containerPort: 2380
      name: server
      protocol: TCP
  restartPolicy: Always

```

```bash
kubectl apply -f etcd.yaml
```
Get the etcd server endpoint:
```bash
$ kubectl get svc -n elastic-job

NAME           TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
etcd-service   ClusterIP   10.96.170.111   <none>        2379/TCP   3h15m
```

### 2. Create a Torch Elastic Job and checkpoint persistent volume
Create a PV and PVC YAML spec that describes the storage dictionary of checkpoint model.
```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-torch-checkpoint
  namespace: elastic-job
spec:
  capacity:
    storage: 5Gi
  volumeMode: Filesystem
  accessModes:
  - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  ...

---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: pvc-torch-checkpoint
  namespace: elastic-job
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 5Gi
  volumeName: pv-torch-checkpoint
  ...
```
```bash
kubectl create -f pv.yaml
```


Create a YAML spec that describes the specifications of a Torch Elastic Job such as the ElasticPolicy, master, worker and volumes like below

```yaml
apiVersion: training.kubedl.io/v1alpha1
kind: "PyTorchJob"
metadata:
  name: "resnet"
  namespace: elastic-job
spec:
  enableElastic: true
  elasticPolicy:
    rdzvBackend: etcd
    rdzvEndpoint: "etcd-service:2379"
    minReplicas: 1
    maxReplicas: 3
    nProcPerNode: 1
  pytorchReplicaSpecs:
    Master:
      replicas: 1
      restartPolicy: ExitCode
      template:
        spec:
          containers:
            - name: pytorch
              image: kubedl/pytorch-dist-example
              imagePullPolicy: Always
    Worker:
      replicas: 1
      restartPolicy: OnFailure
      template:
        spec:
          volumes:
            - name: checkpoint
              persistentVolumeClaim:
                claimName: pvc-torch-checkpoint
          containers:
            - name: pytorch
              image: wanziyu/imagenet:1.1
              imagePullPolicy: Always
              args:
                - "/workspace/examples/imagenet/main.py"
                - "--arch=resnet50"
                - "--epochs=20"
                - "--batch-size=64"
                - "--print-freq=50"
                # number of data loader workers (NOT trainers)
                # zero means load the data on the same process as the trainer
                # this is set so that the container does not OOM since
                # pytorch data loaders use shm
                - "--workers=0"
                - "/workspace/data/tiny-imagenet-200"
                - "--checkpoint-file=/mnt/blob/data/checkpoint.pth.tar"
              resources:
                limits:
                  nvidia.com/gpu: 1
              volumeMounts:
                - name: checkpoint
                  mountPath: "/mnt/blob/data"
```

The `spec.enableElastic` field describes whether user enables the KubeDL torch elastic controller or not. When `enableElastic` field is true and `elasticPolicy` is not empty, the elastic scaling process for this job will be started.

The `spec.elasticPolicy` field specifies the elastic policy including rdzv_backend, rdzv_endpoint, minimum replicas and maximum replicas.
The `rdzvEndpoint` can be set to the etcd service. The `minReplicas` and `maxReplicas` should be set to the desired min and max num nodes (max should not exceed your cluster capacity).

### 3. Submit the Torch Elastic Training Job
```bash
kubectl create -f example/torchelastic/torchelastic-resnet.yaml
```
Check the initial torch elastic training pod status:
```bash
$ kubectl get pod -n elastic-job

NAME                                          READY   STATUS    RESTARTS      AGE
etcd                                          1/1     Running   0             3h56m
resnet-master-0                               1/1     Running   0             12s
resnet-worker-0                               1/1     Running   0             7s
```

Check the initial service status:
```bash
$ kubectl get svc -n elastic-job

NAME              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)     AGE
etcd-service      ClusterIP   10.96.170.111   <none>        2379/TCP    3h57m
resnet-master-0   ClusterIP   None            <none>        23456/TCP   79s
resnet-worker-0   ClusterIP   None            <none>        23456/TCP   74s
```
Check the pytorchJob status:
```bash
$ kubectl describe pytorchjob resnet -n elastic-job

...
Status:
  Conditions:
    Last Transition Time:  2022-09-11T12:08:29Z
    Last Update Time:      2022-09-11T12:08:29Z
    Message:               PyTorchJob resnet is running.
    Reason:                JobRunning
    Status:                True
    Type:                  Running
  Elastic Scaling:
    Worker:
      Continue:           true
      Current Replicas:   1
      Elastic Condition:  Start
      Start Time:         2022-09-11T12:08:54Z
  Replica Statuses:
    Master:
      Active:  1
    Worker:
      Active:  1
```
The `Elastic Scaling` field describes the current elastic scaling status of torch elastic training job. The `Elastic Condition` field indicates whether the elastic scaling workflow continues.

### 3. Watch the elastic scaling process for  torch elastic job
The elastic scaling controller continuously collects the real-time training metrics and decides whether job replicas(ranging from min to max replicas) can be further increased.
If the following conditions satisfy, job will return to the last replicas and the controller will stop the further scaling.

**1. There exists pending pods when scaling in new pods.**
```bash
$ kubectl get pod -n elastic-job

etcd                                        1/1     Running   0             6h
resnet-master-0                             1/1     Running   0             6m53s
resnet-worker-0                             1/1     Running   0             6m47s
resnet-worker-1                             1/1     Running   0             3m44s
resnet-worker-2                             0/1     Pending   0             14s
```
The elastic scaling status is:
```bash
  ...
  Elastic Scaling:
    Worker:
      Continue:           false
      Current Replicas:   2
      Elastic Condition:  Stop
      Last Replicas:      3
      Message:            There exists pending pods, return to the last replicas
      Start Time:         2022-09-11T14:13:55Z
```
The `Message` shows the reason that the elastic scaling process stops.

**2. The training metrics have reached the best values and the job does not need to be further scaled.**

```bash
  ...
  Elastic Scaling:
    Worker:
      Continue:           false
      Current Replicas:   2
      Elastic Condition:  ReachMaxMetric
      Last Replicas:      4
      Message:            Pytorch job has reached the max metrics
      Start Time:         2022-09-11T12:15:24Z
```
Meanwhile, the elastic scaling process will be stopped.



If the elastic scaling process can continue and job replicas can be further increased, the elastic status is as below.

```bash
$ kubectl get pod -n elastic-job

etcd                                        1/1     Running   0             5h57m
resnet-master-0                             1/1     Running   0             3m25s
resnet-worker-0                             1/1     Running   0             3m19s
resnet-worker-1                             1/1     Running   0             16s
```
```bash
 ...
 Elastic Scaling:
    Worker:
      Continue:           true
      Current Replicas:   2
      Elastic Condition:  Continue
      Last Replicas:      1
      Message:            Pytorch job continues to be scaled
      Start Time:         2022-09-11T12:11:54Z
```
Currently, the scaling algorithm is based on the real-time batch training latency collected from running pod logs. The logs of distributed training pods are like as follows.
```bash
Epoch: [17][   0/1563]	Time  5.969 ( 5.969)	Data  0.214 ( 0.214)	Loss 2.1565e+00 (2.1565e+00)	Acc@1  53.12 ( 53.12)	Acc@5  75.00 ( 75.00)
Epoch: [17][  50/1563]	Time  0.258 ( 0.385)	Data  0.155 ( 0.170)	Loss 2.5284e+00 (2.5905e+00)	Acc@1  42.19 ( 39.71)	Acc@5  64.06 ( 65.81)
Epoch: [17][ 100/1563]	Time  0.260 ( 0.323)	Data  0.158 ( 0.164)	Loss 2.4015e+00 (2.6419e+00)	Acc@1  45.31 ( 38.58)	Acc@5  70.31 ( 64.96)
Epoch: [17][ 150/1563]	Time  0.256 ( 0.302)	Data  0.153 ( 0.161)	Loss 2.9381e+00 (2.6560e+00)	Acc@1  34.38 ( 38.15)	Acc@5  64.06 ( 64.82)
Epoch: [17][ 200/1563]	Time  0.296 ( 0.295)	Data  0.189 ( 0.163)	Loss 2.5786e+00 (2.6778e+00)	Acc@1  35.94 ( 37.52)	Acc@5  68.75 ( 64.30)
Epoch: [17][ 250/1563]	Time  0.313 ( 0.291)	Data  0.202 ( 0.165)	Loss 2.6223e+00 (2.6837e+00)	Acc@1  39.06 ( 37.52)	Acc@5  62.50 ( 64.20)
Epoch: [17][ 300/1563]	Time  0.263 ( 0.286)	Data  0.159 ( 0.164)	Loss 2.7830e+00 (2.7005e+00)	Acc@1  40.62 ( 37.18)	Acc@5  57.81 ( 63.83)
Epoch: [17][ 350/1563]	Time  0.267 ( 0.284)	Data  0.163 ( 0.164)	Loss 2.8693e+00 (2.7060e+00)	Acc@1  39.06 ( 37.34)	Acc@5  57.81 ( 63.62)
Epoch: [17][ 400/1563]	Time  0.259 ( 0.281)	Data  0.155 ( 0.163)	Loss 3.0643e+00 (2.7000e+00)	Acc@1  28.12 ( 37.36)	Acc@5  50.00 ( 63.68)
Epoch: [17][ 450/1563]	Time  0.294 ( 0.280)	Data  0.189 ( 0.164)	Loss 2.4482e+00 (2.7056e+00)	Acc@1  43.75 ( 37.21)	Acc@5  70.31 ( 63.57)
```
If you want to change the
training log format in your Python codes, you can revise the  regular expression search formula defined in `log_util.go` to extract the training metrics you specify.