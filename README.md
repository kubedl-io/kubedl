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
- A modular architecture that can be easily extended for more types of DL/ML workloads with shared libraries, see [how to add a custom workload]().

## Getting started

### Prerequisites

1. Install [Golang](https://golang.org/doc/install).

### Installation

1. Download kdl.
```shell
git clone http://github.com/alibaba/kubedl.git
```

2. make
```shell
cd kdl && make
```

### Features

- Support prevalent ML/DL operators in a unified framework.
- GANG scheduling?

## Developing

Here's a brief intro about what a developer must do in order to build a custom operator.

And state what happens step-by-step.


## Community
If you have any questions or want to contribute, GitHub issues or pull requests are warmly welcome.

You can also contact us with the following channels:

## Licensing
