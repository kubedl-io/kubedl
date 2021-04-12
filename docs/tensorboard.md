## TensorBoard

KubeDL can attach a tensorboard to a running tensorflow job.
Users can visualize the tensorflow job with the tensorboard.

To use tensorboard, users must ensure that the tensorflow job logs are created and stored in a kubernetes remote volume (emptyDir, hostPath and local volume are not supported) and the tensorboard pod can mount the volume.

Users can set the tensorboard config in the job's annotation with key `kubedl.io/tensorboard-config` as below.
After that, users can access the tensorboard through this URL `http://<ingress host>/<ingress pathPrefix>/<job namespace>/<job name>`.

For example:
```yaml
    apiVersion: "training.kubedl.io/v1alpha1"
    kind: "TFJob"
    metadata:
      name: "mnist"
      namespace: kubedl
      annotations:
 +      kubedl.io/tensorboard-config: '{"logDir":"/var/log/training","ttlSecondsAfterJobFinished":3600,"ingressSpec":{"host":"locahost","pathPrefix":"/tb"}}'
    spec:
      cleanPodPolicy: None
      tfReplicaSpecs:
        ...
```

A full list of supported options are:

```json
{
    "logDir": "xxx",            // the path of the tensorflow job logs (required).
    "ttlSecondsAfterJobFinished": 3600,     // the TTL to clean up the tensorboard after the job is finished (required).
    "image": "xxx",             // the image of the tensorboard, default value is the job's image (optional).
    "ingressSpec": {            // the ingress of the tensorboard (required).
        "host": "xxx",          // the ingress host (required).
        "pathPrefix": "xxx",    // the pathPrefix will be set to the ingress path with the pattern: <pathPrefix>/<job namespace>/<job name> (required).
        "annotations": {        // the annotations of the ingress (optional).
            "xxx": "xxx"
        }
    }
}
```