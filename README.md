# KubeDL

[![License](https://img.shields.io/badge/license-Apache%202-4EB1BA.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)
[![Build Status](https://travis-ci.com/alibaba/kubedl.svg?branch=master)](https://travis-ci.com/alibaba/kubedl)

KubeDL enables deep learning workloads to run on Kubernetes more easily and efficiently. Its core functionalities include:

- Automatically tunes the best container-level configurations before an ML model is deployed as inference services. - [Morphling Github](https://github.com/alibaba/morphling)
- Model lineage and versioning to track the history of a model natively in CRD: when the model is trained using which data and which image, each version of the model, which version is running etc. 
- Enables storing and versioning a model leveraging container images. Each model version is stored as its own image and can later be served with Serving framework.  
- Support inference frameworks (TensorRT, Triton etc.) and training workloads (Mars, XDL, ElasticDL, Pytorch)in a single unified controller.

Check the website: https://kubedl.io


<div align="center">
 <img src="docs/img/kubedl.png" width="700" title="">
</div> <br/>