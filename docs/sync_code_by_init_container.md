## Sync Code By Init Container

KubeDL supports syncing user code from multiple sources, and mount the code directory into main container. In this way, users
can commit their code to remote source and re-submit training jobs without re-build container images.
So far, git syncing mode has been 

### Git

Users can set their git config by annotation with key `kubedl.io/git-sync-config` as follows: 

```yaml
"kubedl.io/git-sync-config": {...json...}
```

These listed git options are for users to specify and support flexible sync strategies: 

```json
{
"source": "https://github.com/sample/sample.git",  // code source (required).
"image": "xxx",     // image contains toolkits to execute syncing code (optional).
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

And here is a tiny example show you how to submit a Pod with syncing code automatically, and the code dir
will checkout defaulted under your container working dir.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: foo
  annotations:
    kubedl.io/git-sync-config: '{"source": "https://github.alibaba/kubedl" }'
spec:
  containers:
    - image: foo:bar
      name: main
```
