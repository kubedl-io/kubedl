# Run a XDLJob with KubeDL Operator

This tutorial walks you through an example to run a XDLJob.

## Requirements

Before starting this tutorial, you should [install KubeDL Operator](https://github.com/alibaba/kubedl#getting-started) and [enable XDLJob workload](https://github.com/alibaba/kubedl#optional-enable-workload-kind-selectively).

## Install ZooKeeper

XDLJob depends on ZooKeeper to make it's pods to communicate with each other, so we need install ZooKeeper first.

Below installs a single server instance of ZooKeeper.

```bash
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/docs/tutorial/v1/xdl-zk.yaml
```

For production environment, you can follow the [offical tutorial](https://kubernetes.io/docs/tutorials/stateful-application/zookeeper/) to install a three-node ZooKeeper ensemble.

## Run a XDLJob

We need to set ZooKeeper server address and make config file in the XDLJob yaml.For every container in XDLJob, KubeDL operator will attach the environment variables ```TASK_NAME``` and ```TASK_INDEX``` to identify every pods. Also, KubeDL operator will modify the environment variable ```ZK_ADDR``` to add job UUID.

Below run a XDLJob.

```bash
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/docs/tutorial/v1/xdl-job.yaml
```

## Verify XDLJob Started

Check the XDLJob is started, and all pods are Running.

```bash
kubectl get xdljob

NAME                STATE     AGE   FINISHED-TTL   MAX-LIFETIME
xdl-mnist-example   Running   70s   3600

kubectl get po

NAME                            READY   STATUS    RESTARTS   AGE
xdl-mnist-example-ps-0          1/1     Running   0          116s
xdl-mnist-example-scheduler-0   1/1     Running   0          116s
xdl-mnist-example-worker-0      1/1     Running   0          116s
xdl-mnist-example-worker-1      1/1     Running   0          116s
zk-c5cc46c8d-s6bkc              1/1     Running   0          2m26s
```
