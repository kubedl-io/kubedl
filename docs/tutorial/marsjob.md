# Run Mars workloads on kubernetes

## What's Mars

`Mars` is a tensor-based unified framework for large-scale data computation which scales Numpy, Pandas and Scikit-learn, 
see [mars-repo](https://github.com/mars-project/mars) for details. As a data computation framework, `mars` is easy to 
scale out and can run across hundreds of machines simultaneously to accelerate large scale data tasks.<br>

A distributed mars job includes 3 roles to collaborate with each other：
- **WebService**: web-service accepts requests from end-users and forwards the whole tensor-graph to scheduler, it provides a dashboard for end users to track job status and submit tasks interactively.
- **Scheduler**: scheduler compiles and holds a global view of tensor-graph, it schedules 'operands' and 'chunks' to workers.
- **Worker**:  worker listen to 'operands' and 'chunks' dispatched by scheduler, execute the tasks, and reports result back to scheduler.

## Run Distributed Mars Job In Original Way

A distributed `mars` job can run directly by following steps:

1. run `pip install pymars[distributed]` on every node in the cluster to install dependencies needed for distributed execution.
2. start different mars role processes on each node.
    - `mars-scheduler -a <scheduler_ip> -p <scheduler_port>`
    - `mars-web -a <web_ip> -p <web_port> -s <scheduler_ip>:<scheduler_port>`
    - `mars-worker -a <worker_ip> -p <worker_port> -s <scheduler_ip>:<scheduler_port>`
3. usually there must be at least 1 web-service and 1 scheduler and a certain number of workers.
4. after all processes started, users can open the python console run snippet to create a session with web-service and submit tasks.

```python
import mars.tensor as mt
import mars.dataframe as md
from mars.session import new_session
new_session('http://<web_ip>:<web_port>').as_default()
a = mt.ones((2000, 2000), chunk_size=200)
b = mt.inner(a, a)
b.execute()  # submit tensor to cluster
df = md.DataFrame(a).sum()
df.execute()  # submit DataFrame to cluster
```

As it shows, launching a distributed `mars` job manually requires a relatively cognitive load and it can be interrupted easily by the corrupted base environment. 
Moreover, users should handle replicas failover manually which increases the mental burden.

## Run Mars Job On Kubernetes With KubeDL

`KubeDL` extends kubernetes apis and supports running different workloads on it, now `KubeDL` already has the ability to run `mars` workloads
on kubernetes natively. Here are the instructions for users to run a `mars` job.

#### 1. Deploy KubeDL to Cluster

Follow the [installation tutorial](https://github.com/alibaba/kubedl#getting-started) in README and deploy `kubedl` operator to cluster firstly.

#### 2. Apply Mars CRD

`Mars` CRD manifests file describes the structure of a mars job object, you'd apply it to cluster so that its scheme can be 
retrieved by *kube-apiserver* and *kubedl-operator*, run the following command to apply:

```bash
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/config/crd/bases/kubedl.io_marsjobs.yaml
``` 

#### 3. Create a MarsJob Object

In `kubernetes` playground, you're liberated from heavy manual operations and focus on your code, as for mars distributed job, you just need to 
describe a expected state such as resource requests for worker/scheduler, replicas for each role... the expected state is described by a `yaml` file 
and it should be posted to cluster, a typical mars job is shown as below: 

```yaml
apiVersion: kubedl.io/v1alpha1
kind: MarsJob
metadata:
  name: mars-test-demo
  namespace: default
spec:
  cleanPodPolicy: None
  webHost: mars.domain.com
  marsReplicaSpecs:
    Scheduler:
      replicas: 1
      restartPolicy: Never
      template:
        metadata:
          labels:
            mars/service-type: marsscheduler
        spec:
          containers:
            - command:
                - /bin/sh
                - -c
                - python -m mars.deploy.kubernetes.scheduler
              image: mars-image
              imagePullPolicy: Always
              name: mars
              resources:
                limits:
                  cpu: 2
                  memory: 2Gi
                requests:
                  cpu: 2
                  memory: 2Gi
          serviceAccountName: kubedl-sa
    WebService:
      replicas: 1
      restartPolicy: Never
      template:
        metadata:
          labels:
            mars/service-type: marswebservice
        spec:
          containers:
            - command:
                - /bin/sh
                - -c
                - python -m mars.deploy.kubernetes.web
              image: mars-image
              imagePullPolicy: Always
              name: mars
              resources:
                limits:
                  cpu: 2
                  memory: 2Gi
                requests:
                  cpu: 2
                  memory: 2Gi
          serviceAccountName: kubedl-sa
    Worker:
      replicas: 2
      restartPolicy: Never
      template:
        metadata:
          labels:
            mars/service-type: marsworker
        spec:
          containers:
            - command:
                - /bin/sh
                - -c
                - python -m mars.deploy.kubernetes.worker
              image: mars-image
              imagePullPolicy: Always
              name: mars
              resources:
                limits:
                  cpu: 2
                  memory: 2Gi
                requests:
                  cpu: 2
                  memory: 2Gi
          serviceAccountName: kubedl-sa
status: {}
```

The `spec` field embedded in job object describes the expected state of each replica, including `replicas`, `restartPolicy`, `template`...and its current observed
state will be recorded in `status` filed. Run following command to start a example mars job:

```bash
kubectl create -f example/mars/mars-test-demo.yaml
```

then retrieve latest mars job status by running following commands:

```bash
$ k get marsjob
NAME             STATE     AGE   FINISHED-TTL   MAX-LIFETIME
mars-test-demo   Running   40m
$ k get po
NAME                                            READY   STATUS             RESTARTS   AGE
mars-test-demo-scheduler-0                      1/1     Running            0          40m
mars-test-demo-webservice-0                     1/1     Running            0          40m
mars-test-demo-worker-0                         1/1     Running            0          40m
mars-test-demo-worker-1                         1/1     Running            0          40m

``` 

#### 4. Access web-service.

<div align="center">
 <img src="../img/mars-ingress.png" width="700" title="Mars-Ingress">
</div> <br/>

Web service visualizes job status, computation process progress and provides an entry for interactive submission, however, web service
instance was running inside a kubernetes cluster which may have network connectivity issues with external users, so `KubeDL` provides 
two access mode for users in different network environment.

##### 4.1 Access web-service in-cluster.

For users in the same network environment with web service instance, they can directly access its *service* without any other additional configurations,
and the address is formatted as: `{webservice-name}.{namespace}`, it is a `A` record generated by `CoreDNS`, so you have to ensure that `CoreDNS` has been
deployed.

##### 4.2 Access web-service out-of-cluster.

For users in different network environment(e.g. an internet user wants to access a mars web-service running in vpc), users have to apply an SLB address first
, so that they can ping the ip in **vpc** with a public address by SLB domain resolving, then in job spec, users just need fill the `spec.webHost` field with 
their applied SLB address, `KubeDL`will generated ingress instance with routing rules, so that external traffic can be routed to target web service and 
becomes available for out-of-cluster users.

#### 5. Memory Tuning Policy

`Worker` as the replica type that actually performs computing tasks in `MarsJob`, it has to deal with various scenarios with diverse memory demands, for 
example, you may want to swap cold in-memory data out to spill dirs and persist in ephemeral-storage. `Mars` provides plentiful memory tuning options which
has been integrated to `MarsJob` type definition, including following ones:

- plasmaStore: PlasmaStore specify the socket path of plasma store that handles shared memory between all worker processes.
- lockFreeFileIO: LockFreeFileIO indicates whether spill dirs are dedicated or not.
- spillDirs: SpillDirs specify multiple directory paths, when size of in-memory objects is about to reach the limitation, mars workers will swap cold data out to spill dirs and persist in ephemeral-storage.
- workerCachePercentage: WorkerCachePercentage specify the percentage of total available memory size can be used as cache, it will be overridden by workerCacheSize if it is been set.
- workerCacheSize：WorkerCacheSize specify the exact cache quantity can be used.

users can set above options in `job.spec.memoryTuningPolicy` field: 

```yaml
apiVersion: kubedl.io/v1alpha1
kind: MarsJob
metadata:
  name: mars-test-demo
  namespace: default
spec:
  cleanPodPolicy: None
  memoryTuningPolicy:
    plasmaStore: string              # /etc/pstore/...
    lockFreeFileIO: bool             # false
    spillDirs: []string              # ...
    workerCachePercentage: int32     # 80, indicates 80%
    workerCacheSize: quantity        # 10Gi
  marsReplicaSpecs:
    ...
```
