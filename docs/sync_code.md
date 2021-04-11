## Download Custom Artifacts 

KubeDL supports downloading custom artifacts from remote repositories into the container. With that, user
can commit their code to remote repository and re-submit the jobs without re-building the image to include the artifacts.

Currently, only github is supported. The framework is pluggable and can easily support other storage systems like `HDFS`. 

### Github

Users can set the git config in the job's annotation with key `kubedl.io/git-sync-config` as below. The git repo will be 
downloaded and saved in the container's `working dir` by default. Please use the git repo's clone url ending with the `.git`,
rather than the git repo's web url.

```yaml
    apiVersion: training.kubedl.io/v1alpha1
    kind: "TFJob"
    metadata:
      name: "mnist"
      namespace: kubedl 
      annotations:
 +      kubedl.io/git-sync-config: '{"source": "https://github.com/alibaba/kubedl.git" }'
    spec:
      cleanPodPolicy: None 
      tfReplicaSpecs:
        ...
```


In addition, there are various of options for users to customize downloading policies, 
a full list of supported options are as follows:

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
