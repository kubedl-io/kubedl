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
- Instrumented with rich prometheus metrics to provide more insights about the job status, such as job launch delay, total number of running jobs.
- Support gang scheduling with a pluggable interface to support different backend gang schedulers.
- Support debugging  workload types selectively.
- Support running a job (in the form of YAML) with source code from github/remote store(e.g. hdfs) without rebuilding the image
- A modular architecture that can be easily extended for more types of DL/ML workloads with shared libraries, see [how to add a custom workload](https://github.com/alibaba/kubedl/blob/master/docs/how-to-add-a-custom-workload.md).

## Getting started

### Installation

#### Download and build

```bash
git clone http://github.com/alibaba/kubedl.git
cd kubedl && make
```

#### Install CRDs

```bash
kubectl apply -f http://github.com/alibaba/config/crd/bases
```

#### Deploy KubeDL as a deployment

```bash
kubectl apply -f http://github.com/alibaba/config/manager/all_in_one.yaml
```

The official KubeDL image is hosted under [docker hub](https://hub.docker.com/r/kubedl/kubedl).

## Example

This example demonstrates how to run a simple MNist Tensorflow job with KubeDL.
- Install CRD for TensorFlow job
  `kubectl apply -f kubedl/example/crd/crd-tf.yaml`
- Deploy the event volume
  `kubectl apply -f kubedl/example/tf/tfevent-volume`
- Submit the TFJob
  `kubectl apply -f tf_job_mnist.yaml`
- Monitor the status of the Tensorflow job
  `kubectl describe tfjob mnist`
- Delete the job
  `kubectl delete tfjob mnist`

## Developer Guide

Th Makefile in the root folder  describes the options to build and install. Commonly used options include:

- Build the controller manager binary: 
  `make manager`
- Run the tests: 
  `make test`
- Generate manifests e.g. CRD, RBAC YAML files etc: 
  `make manifests`

To build the docker image, named as `alibaba/kubedl:v1alpha1` by default:

```bash
export IMG=<your_image_name> && make docker-build
```

To push the image: 

```bash
export IMG=<your_image_name> && make docker-push 
// or
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
