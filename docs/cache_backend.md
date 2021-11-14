# CacheBackend

## Background
A deep learning job usually consists of data preprocessing, data loading to GPU memory, and model training. In these processes, I/O of datasets is one of the bottlenecks affecting the training time of deep learning jobs. Improving the access speed of datasets can reduce the time spent in training, improve the utilization rate of GPU and the efficiency of model training.

Kubedl supports caching datasets using a third-party caching engine. Based on this feature, the efficiency of users using KubeDL for deep learning job can be improved. 

## How To Use

At present, KubeDL only supports [Fluid](https://github.com/fluid-cloudnative/fluid) as the caching engine, but the framework of this feature is plug-in. In the future, it can support other data caching applications.

### Fluid

In order to use Fluid as the cache backend to speed up access to the dataset, you need to add some fields to job spec. At the same time, you also need to deploy Fluid in your Kubernetes cluster in advance to provide cache support. Please refer to [here](https://github.com/fluid-cloudnative/fluid/blob/master/docs/en/userguide/install.md) for specific deployment steps

#### An Example

To demonstrate how to use the cache feature, here is a demo tfjob:

```yaml
apiVersion: "training.kubedl.io/v1alpha1"
kind: "TFJob"
metadata:
  name: "tf-cache"
spec:
  cleanPodPolicy: None  
  cacheBackend:
    mountPath: "/data"   # mountPath is the path to which cached files are mounted to the container
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

This example uses the [MNIST dataset](http://yann.lecun.com/exdb/mnist/), which is a very classic dataset in the field of computer vision. If you want to practice this case yourself, you need to download the MNIST dataset and replace the file path at `spec.cachebackend.cacheengine.fluid.dataset.datasources.path`. Note that the prefix `local://` is reserved, which means to load the source file locally (Or you can replace it with any path that can be recognized by alluxio).

Dataset and alluxioruntime are the basic components of Fluid. These two abstractions are used to define dataset and configure cache parameters.The parameter meanings of these two parts are almost the same as those in Fluid (some names may be modified in KubeDL to make naming easier to understand)

#### Create Job and Check Status

Create Job

```shell
$ kubectl apply -f tf_job_mnist_cache.yaml 
tfjob.training.kubedl.io/tf-cache created
```

If enabled cache, KubeDL will first create a cachebackend

```shell
$ kubectl get cacheBackend
NAME                   JOB-NAME   CACHE-STATUS    AGE
cache-tf-cache-9a79c   tf-cache   CacheCreating   1s
```

After that, the status will change to `PVCCreating`. It means that cachebackend is already requesting the Fluid to create dataset and alluxioruntime

```shell
$ kubectl get cacheBackend
NAME                   JOB-NAME   CACHE-STATUS   AGE
cache-tf-cache-9a79c   tf-cache   PVCCreating    3s
```

After a while, can view the status of the dataset and the alluxio runtime. At this moment, the dataset has not been cached

```shell
$ kubectl get dataset
NAME                   UFS TOTAL SIZE   CACHED   CACHE CAPACITY   CACHED PERCENTAGE   PHASE      AGE
cache-tf-cache-9a79c   11.06MiB                                                       NotBound   53s

$ kubectl get alluxioruntime
NAME                   MASTER PHASE   WORKER PHASE   FUSE PHASE   AGE
cache-tf-cache-9a79c   Ready          Ready          Ready        59s
```

Kubedl will automatically check whether the PVC is created. If the status of cachebackend is updated to `PVCCreated`, it indicates that the PVC created by Fluid for us has been generated

```shell
$ kubectl get cacheBackend
NAME                   JOB-NAME   CACHE-STATUS   AGE
cache-tf-cache-9a79c   tf-cache   PVCCreated     92s
```

Kubedl will not run the job until PVC is created. When the job is running, PVC has been mounted into the containers

```shell
$ kubectl get po
NAME                                READY   STATUS    RESTARTS   AGE
cache-tf-cache-9a79c-fuse-z98qb     1/1     Running   0          62s
cache-tf-cache-9a79c-master-0       2/2     Running   0          87s
cache-tf-cache-9a79c-worker-4cq44   2/2     Running   0          62s
tf-cache-worker-0                   1/1     Running   0          14s
```

Viewing the status of the dataset, can see that `CACHED PERCENTAGE = 100.0%`, which means that the dataset used in this job has been cached

```shell
$ kubectl get dataset
NAME                   UFS TOTAL SIZE   CACHED     CACHE CAPACITY   CACHED PERCENTAGE   PHASE   AGE
cache-tf-cache-9a79c   11.06MiB         11.06MiB   1.00GiB          100.0%              Bound   118s
```

Job complete

```shell
$ kubectl get po
NAME                                READY   STATUS      RESTARTS   AGE
cache-tf-cache-9a79c-fuse-z98qb     1/1     Running     0          113s
cache-tf-cache-9a79c-master-0       2/2     Running     0          2m18s
cache-tf-cache-9a79c-worker-4cq44   2/2     Running     0          113s
tf-cache-worker-0                   0/1     Completed   0          65s
```

#### Clean Up

Delete job

```shell
$ kubectl delete -f tf_job_mnist_cache.yaml
tfjob.training.kubedl.io "tf-cache" deleted
```

Delete dataset

```shell
$ kubectl delete dataset cache-tf-cache-9a79c       
dataset.data.fluid.io "cache-tf-cache-9a79c" deleted
```

Delete alluxioruntime

```shell
$ kubectl delete alluxioruntime cache-tf-cache-9a79c                                                   
alluxioruntime.data.fluid.io "cache-tf-cache-9a79c" deleted
```

