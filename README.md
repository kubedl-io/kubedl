# KubeDL

[![License](https://img.shields.io/badge/license-Apache%202-4EB1BA.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubedl-io%2Fkubedl.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubedl-io%2Fkubedl?ref=badge_shield)
[![KubeDL Action Status](https://github.com/kubedl-io/kubedl/workflows/CI/badge.svg)](https://github.com/kubedl-io}/kubedl}/actions)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5072/badge)](https://bestpractices.coreinfrastructure.org/projects/5072)


<h1 align="center">
    <img src="./docs/img/kubedllogo.png" alt="logo" width="400">
</h1>

KubeDL enables deep learning workloads to run on Kubernetes more easily and efficiently. 
<div align="left">
 <img src="https://v6d.io/_static/cncf-color.svg" width="400" title="">
</div> <br/>

KubeDL is a [CNCF sandbox](https://www.cncf.io/sandbox-projects/) project.


Its core functionalities include:

- Automatically tunes the best container-level configurations before an ML model is deployed as inference services. - [Morphling Github](https://github.com/alibaba/morphling)
- Model lineage and versioning to track the history of a model natively in CRD: when the model is trained using which data and which image, each version of the model, which version is running etc. 
- Enables storing and versioning a model leveraging container images. Each model version is stored as its own image and can later be served with Serving framework.  
- Support inference frameworks and training workloads (Tensorflow, Pytorch. [Mars](https://github.com/mars-project/mars) etc.)in a single unified controller.

Check the website: https://kubedl.io


<div align="center">
 <img src="docs/img/kubedl.png" width="700" title="">
</div> <br/>



## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubedl-io%2Fkubedl.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubedl-io%2Fkubedl?ref=badge_large)
