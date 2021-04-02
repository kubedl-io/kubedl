# KubeDL

[![License](https://img.shields.io/badge/license-Apache%202-4EB1BA.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)
[![Build Status](https://travis-ci.com/alibaba/kubedl.svg?branch=master)](https://travis-ci.com/alibaba/kubedl)

KubeDL is short for **Kube**rnetes-**D**eep-**L**earning. It is a unified operator that supports running
multiple types of distributed deep learning/machine learning workloads on Kubernetes. Check the website: https://kubedl.io


<div align="center">
 <img src="docs/img/stack.png" width="700" title="Stack">
</div> <br/>

Currently, KubeDL supports the following ML/DL jobs:

- [TensorFlow](https://github.com/tensorflow/tensorflow)
- [PyTorch](https://github.com/pytorch/pytorch)
- [XGBoost](https://github.com/dmlc/xgboost)
- [XDL](https://github.com/alibaba/x-deeplearning/)
- [Mars](https://github.com/mars-project/mars)
- MPI Job


## Features
- Support running prevalent DeepLearning workloads in a single operator.
- Support running jobs with [custom artifacts from remote repository](./docs/sync_code.md ) such as github, saving users from manually baking the artificats into the image. 
- Instrumented with unified [prometheus metrics](./docs/metrics.md)  for different types of DL jobs, such as job launch delay, number of pending/running jobs.
- Support job metadata persistency with a pluggable storage backend such as Mysql.
- Provide more granular information on kubectl command line to show job status.
- Support advanced scheduling features such as gang scheduling with pluggable backend schedulers.
- A modular architecture that can be easily extended for more types of DL/ML workloads with shared libraries, see [how to add a custom job workload](https://github.com/alibaba/kubedl/blob/master/docs/how-to-add-a-custom-workload.md).
- Run jobs with Host network.
### Build right away

```bash
make manager
```
### Run the tests

```bash
make test
```
### Generate manifests e.g. CRD, RBAC YAML files etc

```bash
make manifests
```
### Build the docker image

```bash
export IMG=<your_image_name> && make docker-build
```

### Push the image

```bash
docker push <your_image_name>
```

Check the `Makefile` in the root directory for more details.
