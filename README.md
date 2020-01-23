# KubeDL

KubeDL is short for **Kube**rnetes-**D**eep-**L**earning. It is a unified operator that supports running
multiple types of distributed deep learning/machine learning workloads on Kubernetes.

Currently, KubeDL supports the following types of ML/DL jobs:

- [TensorFlow](https://github.com/tensorflow/tensorflow)
- [PyTorch](https://github.com/pytorch/pytorch)
- [XGBoost](https://github.com/dmlc/xgboost)
- [XDL](https://github.com/alibaba/x-deeplearning/)

KubeDL is API compatible with [tf-operator](https://github.com/kubeflow/tf-operator), [pytorch-operator](https://github.com/kubeflow/pytorch-operator),
[xgboost-operator](https://github.com/kubeflow/xgboost-operator) and integrates them with enhanced features as below:

- Support running prevalent ML/DL workloads in a single operator.
- Instrumented with rich prometheus [metrics](./docs/metrics.md) to provide more insights about the job stats, such as job launch delay, current number of pending/running jobs.
- Enable specific workload type according to the installed CRDs automatically or through the startup flags explicitly.
- Support running a job (in the form of YAML) with [source code from github](./docs/sync_code.md ) without rebuilding the image.
- A modular architecture that can be easily extended for more types of DL/ML workloads with shared libraries, see [how to add a custom job workload](https://github.com/alibaba/kubedl/blob/master/docs/how-to-add-a-custom-workload.md).
- Support gang scheduling with a pluggable interface to support different backend gang schedulers.

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

#### Optional: Enable workload kind selectively

If you only need some of the workload types and want to disable others, you can use either one of the three options or all of them:

- Set env `WORKLOADS_ENABLE` in KubeDL container when you do deploying. The value is a list of workloads types that you want to enable. For example, `WORKLOADS_ENABLE=TFJob,PytorchJob` means only TFJob and PytorchJob workload are enabled, the others are disabled.

- Set startup arguments `--workloads` in KubeDL container args when you do deploying. The value configuration is consistent with `WORKLOADS_ENABLE` env. 

- **[DEFAULT]** Only install the CRDs you need, KubeDL will automatically enables corresponding workload controllers, you can set `--workloads auto` or `WORKLOADS_ENABLE=auto` explicitly. This is a recommended approach.



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
Check the [documents](docs/metrics.md) for the prometheus metrics supported for KubeDL operator.

## Code Sync
KubeDL supports running jobs with source code synced from remote source dynamically without rebuilding the image.
Check the [documents](docs/sync_code.md) for details.

## Developer Guide

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

- Slack: [Join the Channel](https://join.slack.com/t/kubedl/shared_invite/enQtOTAwNjI5NjUyNjMxLWU3N2UxMzdjZWQ0YTc3MzE1NWUxZWU1MmVkMmZhNWIxOTUyZDc1OWVhMTA4NmRkYmVjMzkxNTllNGY4NGYwZTc)

- Dingtalk Group(钉钉讨论群)

<div align="center">
  <img src="docs/img/kubedl.JPG" width="250" title="dingtalk">
</div>

## Copyright

Certain implementations rely on existing code from the Kubeflow community and the credit goes to original Kubeflow authors.
