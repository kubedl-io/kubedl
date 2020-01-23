## Code Sync

KubeDL supports syncing user code from multiple sources, and mount the code directory into main container. In this way, users
can commit their code to remote source and re-submit training jobs without re-build container images.

As of now, git-sync mode is supported.

### Git

Users can set the git config in the job's annotation with key `kubedl.io/git-sync-config` as follows: 

```yaml
    apiVersion: "kubeflow.org/v1"
    kind: "TFJob"
    metadata:
      name: "mnist"
      namespace: kubedl 
      annotations:
 +      kubedl.io/git-sync-config: '{"source": "https://github.alibaba/kubedl" }'
    spec:
      cleanPodPolicy: None 
      tfReplicaSpecs:
        ...
```

The git repo will be downloaded and saved in the container's working dir by default.

A full list of supported options are:

```json
{
    "source": "https://github.com/sample/sample.git",  // code source (required).
    "image": "xxx",     // the image to execute the git-sync logic (optional).
    "rootPath": "xxx",  // the path to save downloaded files (optional).
    "destPath": "xxx",  // the name of (a symlink to) a directory in which to check-out files (optional).
    "envs": [],         // user-customized environment variables (optional).
    "branch": "xxx",    // git repo branch (optional).
    "revison": "xxx",   // git repo commit revision (optional).
    "depth": "xxx",     // git sync depth (optional).
    "maxFailures" : 3,  // max consecutive failures allowed (optional).
    "ssh": false,       // use ssh mode or not (optional).
    "sshFile": "xxx",   // ssh file path (optional).
    "user": "xxx",      // git config username (optional).
    "password": "xxx"   // git config password (optional).
}
```
