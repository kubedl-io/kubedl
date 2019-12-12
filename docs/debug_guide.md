# How to DEBUG

## DEBUG: start with process locally

**Credentials**

To run KubeDL locally, you must have the access to the kubernetes cluster, the credential is a distributed
kube-config cert file.

**Install CRDs and run KubeDL-manager**

Run the following commands

```bash
export KUBECONFIG=${PATH_TO_CONFIG}
// or specify the path by --kubeconfig {PATH_TO_CONFIG}
make install
make run
```

KubeDL supports debugging workloads selectively. You can enable a specific workload by parsing the 
parameter `--workloads {workload-to-debug}` while starting KubeDL. Check the printed logs to see 
if the job controller is started as expected.

## DEBUG: start with Pod

The followings are the steps to debug KubeDL manager using Pod.

**Install docker**

Following the [official docker installation guide](https://docs.docker.com/install/).

**Install minikube**

Follow the [official minikube installation guide](https://kubernetes.io/docs/tasks/tools/install-minikube/).

**Develop customized controller manager**

Make your own code changes and validate the build by running `make manager` in KubeDL directory.

**Deploy customized controller manager**

Your customized controller manager will be deployed as a replicaset to replace the original KubeDL controller manager.
The deployment can be done via the following steps assuming a clean environment:

* Prerequisites: create new/use existing [dock hub](https://hub.docker.com/) account ($DOCKERID), and create a `kubedl` repository in it. Also, [install KubeDL CRDs](../README.md#install-crds);
* step 1: `docker login` with the $DOCKERID account;
* step 2: `export IMG=<image_name>` to specify the target image name. e.g., `export IMG=$DOCKERID/kubedl:test`;
* step 3: `make docker-build` to build the image locally;
* step 4: `make docker-push` to push the image to dock hub under the `kubedl` repository;
* step 5: change the `config/manager/all_in_one.yaml` and replace the container image of the controller manager statefulset to `$DOCKERID/kubedl:test`

```yaml
spec:
      containers:
        - command:
            - /manager
          args:
            - "--workloads {workload-to-debug}"
          image: $DOCKERID/kubedl:test
          imagePullPolicy: Always
          name: kubedl
```

* step 6: `kubectl delete rs kubedl-controller-manager -n kubedl` to remove the old replicaset if any;
* step 7: `kubectl apply -f config/manager/all_in_one.yaml` to install the new replicaset with the customized controller manager image;

You can now perform manual tests and use `kubectl logs kubedl-controller-manager-0 -n kubedl-system` to check controller logs for debugging.