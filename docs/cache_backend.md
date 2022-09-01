# CacheBackend

## Background

A deep learning job usually consists of data preprocessing, data loading to GPU memory, and model training. In these processes, I/O of datasets is one of the bottlenecks affecting the training time of deep learning jobs. Improving the access speed of datasets can reduce the time spent in training, improve the utilization rate of GPU and the efficiency of model training.

Kubedl supports caching datasets using a third-party caching engine. Based on this feature, the efficiency of users using KubeDL for deep learning job can be improved.

## How To Use

At present, KubeDL only supports [Fluid](https://github.com/fluid-cloudnative/fluid) as the caching engine, but the framework of this feature is plug-in. In the future, it can support other data caching applications.

### Fluid

In order to use Fluid as the cache backend to speed up access to the dataset, you need to add some fields to job spec. At the same time, you also need to deploy Fluid in your Kubernetes cluster in advance to provide cache support. Please refer to [here](https://github.com/fluid-cloudnative/fluid/blob/master/docs/en/userguide/install.md) for specific deployment steps

#### Check CacheBackend

To demonstrate how to use the cache feature, here is a demo CacheBackend:

```yaml
apiVersion: "cache.kubedl.io/v1alpha1"
kind: "CacheBackend"
metadata:
  name: "test-cachebackend"
spec:
  dataset:
    dataSources:
      - location: local:///dataset/mnist  # path property can be any legal UFS path acknowledged by Alluxio
        subDirName: mnist   # dataset needs to specify the name of the file directory in the mount path
  cacheEngine:
    fluid:
      alluxioRuntime:
        replicas: 1
        tieredStorage:
          - cachePath: /dev/shm
            quota: "1Gi"
            mediumType: MEM
  options:
    idleTime: 60
```

This example uses the [MNIST dataset](http://yann.lecun.com/exdb/mnist/), which is a very classic dataset in the field of computer vision. If you want to practice this case yourself, you need to download the MNIST dataset and replace the file path at `spec.cachebackend.cacheengine.fluid.dataset.datasources.path`. Note that the prefix `local://` is reserved, which means to load the source file locally (Or you can replace it with any path that can be recognized by alluxio).

Dataset and AlluxioRuntime are the basic components of Fluid. These two abstractions are used to define dataset and configure cache parameters.The parameter meanings of these two parts are almost the same as those in Fluid (some parameters may be modified in KubeDL to make naming easier to understand)

CacheBackend supports simple parameters to support more functionality, it mainly using the `options` field.  In the above demo file, we added the `idleTime` to `options`. `idleTime` controls the maximum unused time of a CacheBackend. If a CacheBackend has been unused for more than Idletime, the controller will automatically remove the infrequently used CacheBackend.

#### Create CacheBackend

The CacheBackend mentioned in the previous section can be found in `example/cachebackend/cache.yaml`.

```shell
$ kubectl apply -f example/cachebackend/cache.yaml
cachebackend.cache.kubedl.io/test-cachebackend created
```

Check the status of CacheBackend. If enabled cache, KubeDL will create a cachebackend. The status is `PVCCreating`, which means that cachebackend is already requesting the Fluid to create dataset and alluxioruntime.

The `LAST-USED-TIME` means that current CacheBackend is in inactive status and it is equal to the completion time of the last job that used CacheBackend. The `USED-NUM` is the number of jobs that are currently using CacheBackend.

```shell
$ kubectl get cacheBackend
NAME                ENGINE   STATUS        USED-NUM   LAST-USED-TIME   AGE
test-cachebackend   fluid    PVCCreating                               28s
```

After a while, can view the status of the dataset and the alluxio runtime. At this moment, the dataset has not been cached.

```shell
$ kubectl get dataset
NAME                UFS TOTAL SIZE   CACHED   CACHE CAPACITY   CACHED PERCENTAGE   PHASE   AGE
test-cachebackend   11.06MiB         0.00B    1.00GiB          0.0%                Bound   75s

$ kubectl get alluxioruntime
NAME                MASTER PHASE   WORKER PHASE   FUSE PHASE   AGE
test-cachebackend   Ready          Ready          Ready        56s
```

Kubedl will automatically check whether the PVC is created. If the status of cachebackend is updated to `PVCCreated`, it indicates that the PVC created by Fluid for us has been generated.

```shell
$ kubectl get cacheBackend  
NAME                ENGINE   STATUS       USED-NUM   LAST-USED-TIME   AGE
test-cachebackend   fluid    PVCCreated              5s               88s

$ kubectl get pvc
NAME                STATUS   VOLUME                      CAPACITY   ACCESS MODES   STORAGECLASS   AGE
test-cachebackend   Bound    default-test-cachebackend   100Gi      ROX            fluid          52s
```

#### Check Job

Next we will show how to use CacheBackend to speed up jobs in Kubedl.

To enable CacheBackend, you need to add a CacheBackend object to the Spec field of your job ( line 7 ~ 9 ). It only need to specify which cacheBackend you want to use and which directory you want to mount in Pods. The Controller will automatically bind the PersistVolumeClaim and mount the Volume later.

```yaml
apiVersion: "training.kubedl.io/v1alpha1"
kind: "TFJob"
metadata:
  name: "tf-cache"
spec:
  cleanPodPolicy: None
+ cacheBackend:
+   name: test-cachebackend
+   mountPath: "/data"   # mountPath is the path to which cached files are mounted to the container
  tfReplicaSpecs:
    Worker:
      replicas: 1 
      restartPolicy: Never
      template:
        spec:
          containers:
            - name: tensorflow
              image: kubedl/tf-mnist-with-summaries:1.0
              command:
                - "python"
                - "/var/tf_mnist/mnist_with_summaries.py"
                - "--log_dir=/train/logs"
                - "--data_dir=/data/mnist"  # dataset directory, equivalent to "mountPath + subdirName"
                - "--learning_rate=0.01"
                - "--batch_size=150"
              resources:
                limits:
                  cpu: 2048m
                  memory: 2Gi
                requests:
                  cpu: 1024m               
                  memory: 1Gi
```

#### Create Job

```shell
$ kubectl apply -f tf_job_mnist_cache.yaml 
tfjob.training.kubedl.io/tf-cache created
```

Check CacheBackend in job status

```shell
$ kubectl get tfjob                                  
NAME       STATE     AGE   MODEL-VERSION   CACHE-BACKEND       MAX-LIFETIME   TTL-AFTER-FINISHED
tf-cache   Running   8s                    test-cachebackend                  
```

Kubedl will not run the job until PVC is created. When the job is running, PVC has been mounted into the containers

```shell
$ kubectl get po    
NAME                           READY   STATUS    RESTARTS   AGE
tf-cache-worker-0              1/1     Running   0          20s
```

Viewing the status of the dataset, can see that `CACHED PERCENTAGE = 100.0%`, which means that the dataset used in this job has been cached

```shell
$ kubectl get dataset
NAME                UFS TOTAL SIZE   CACHED     CACHE CAPACITY   CACHED PERCENTAGE   PHASE   AGE
test-cachebackend   11.06MiB         11.06MiB   1.00GiB          100.0%              Bound   2m45s
```

Job complete

```shell
$ kubectl get po
NAME                           READY   STATUS      RESTARTS   AGE
tf-cache-worker-0              0/1     Completed   0          2m18s
```

#### Clean Up

Delete Job

```shell
$ kubectl delete -f tf_job_mnist_cache.yaml
tfjob.training.kubedl.io "tf-cache" deleted
```

Delete CacheBackend.

```shell
$ kubectl delete cacheBackend test-cachebackend
cachebackend.cache.kubedl.io "test-cachebackend" deleted
```

After CacheBackend removed, the `dataset`, `alluxioruntime` and `pvc` will be automatically deleted.

## Cache Reuse

CacheBackend objects have job-independent life cycles. When a job bound to CacheBackend is deleted, CacheBackend and cached datasets will remain in the environment. It means that you can bind multiple jobs to the same CacheBackend or bind the same cacheBackend to different jobs at different times.
