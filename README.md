# KubeDL

> A unified framework for deep-learning/machine-learning operators on Kubernetes.


KubeDL is short for **Kube**rnetes-**D**eep-**L**earning. It is a unified operator that supports running
multiple types of distributed deep learning/machine learning workloads on Kubernetes.

Currently, KubeDL supports the following types of ML/DL jobs:

1. [Tensorflow](https://github.com/tensorflow/tensorflow)
2. [Pytorch](https://github.com/pytorch/pytorch)
3. [XGBoost](https://github.com/dmlc/xgboost)
4. [XDL](https://github.com/alibaba/x-deeplearning/tree/master/xdl/xdl)

KubeDL is API compatible with [tf-operator](https://github.com/kubeflow/tf-operator), [pytorch-operator](https://github.com/kubeflow/pytorch-operator),
[xgboost-operator](https://github.com/kubeflow/xgboost-operator) and integrates them with enhanced features as below:

- Support running prevalent ML/DL workloads in a single operator.
- Instrumented with rich prometheus [metrics](./docs/metrics.md) to provide more insights about the job stats, such as job launch delay, current number of pending/running jobs.
- Support gang scheduling with a pluggable interface to support different backend gang schedulers.
- Enable specific job workload type selectively.
- Support running a job (in the form of YAML) with source code from github/remote store(e.g. hdfs) without rebuilding the image
- A modular architecture that can be easily extended for more types of DL/ML workloads with shared libraries, see [how to add a custom job workload](https://github.com/alibaba/kubedl/blob/master/docs/how-to-add-a-custom-workload.md).

## Getting started

#### Install CRDs

```bash
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/config/crd/bases/kubeflow.org_pytorchjobs.yaml
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/config/crd/bases/kubeflow.org_tfjobs.yaml
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/config/crd/bases/xdl.alibaba.com_xdljobs.yaml
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/config/crd/bases/xgboostjob.kubeflow.org_xgboostjobs.yaml
```

#### Deploy KubeDL as a deployment

```bash
kubectl apply -f https://raw.githubusercontent.com/alibaba/kubedl/master/config/manager/all_in_one.yaml
```

The official KubeDL operator image is hosted under [docker hub](https://hub.docker.com/r/kubedl/kubedl).

## Run an Example Job 

This example demonstrates how to run a simple MNist Tensorflow job with KubeDL.

#### Submit the TFJob

```bash
kubectl apply -f http://raw.githubusercontent.com/alibaba/kubedl/example/tf/tf_job_mnist.yaml
```

#### Monitor the status of the Tensorflow job

```bash
kubectl describe tfjob mnist -n kubedl
```

#### Delete the job

```bash
kubectl delete tfjob mnist -n kubedl
```

## Metrics
See the [details](docs/metrics.md) for the prometheus metrics supported for KubeDL operator.

## Developer Guide

#### Download 

```bash
git clone http://github.com/alibaba/kubedl.git
```

#### Build the controller manager binary

```bash
make manager
```
#### Run the tests

```bash
make test
```
#### Generate manifests e.g. CRD, RBAC YAML files etc

```bash
make manifests
```
#### Build the docker image

```bash
export IMG=<your_image_name> && make docker-build
```

#### Push the image

```bash
docker push <your_image_name>
```

To develop/debug KubeDL controller manager locally, please check the [debug guide](https://github.com/alibaba/kubedl/blob/master/docs/debug_guide.md).

## Community

If you have any questions or want to contribute, GitHub issues or pull requests are warmly welcome.
You can also contact us via the following channels:

Slack:

Dingtalk:

## Copyright

Certain implementations rely on existing code from the Kubeflow community and the credit goes to original Kubeflow authors.
