# How to DEBUG

## DEBUG: start with process locally

- Credentials

    To run KubeDL locally, you must have the access to the kubernetes cluster, the credential is a distributed
    kube-config cert file.

- Install CRDs and run KubeDL operator
    
    ```bash
    export KUBECONFIG=${PATH_TO_CONFIG}
    // or specify the path by --kubeconfig {PATH_TO_CONFIG}
    make install
    make run
    ```

KubeDL supports running workloads selectively. You can enable a specific workload by parsing the 
parameter `--workloads {workload-to-debug}` while starting KubeDL. Check the printed logs to see 
if the job controller is started as expected.

## DEBUG: start with Pod

The followings are the steps to debug KubeDL operator using Pod.

- Install docker

  Following the [official docker installation guide](https://docs.docker.com/install/).

- Install minikube**

  Follow the [official minikube installation guide](https://kubernetes.io/docs/tasks/tools/install-minikube/).

- Develop customized KubeDL operator

  Make your own code changes and validate the build by running `make manager` in KubeDL directory.

- Deploy customized operator

  Your customized operator will be deployed as a replicaset to replace the original KubeDL operator.
  The deployment can be done via the following steps assuming a clean environment:

    * Prerequisites: create new/use existing [dock hub](https://hub.docker.com/) account ($DOCKERID), and create a `kubedl` repository in it. Also, [install KubeDL CRDs](../README.md#install-crds);
    * step 1: `docker login` with the $DOCKERID account;
    * step 2: `export IMG=<image_name>` to specify the target image name. e.g., `export IMG=$DOCKERID/kubedl:test`;
    * step 3: `make docker-build` to build the image locally;
    * step 4: `make docker-push` to push the image to dock hub under the `kubedl` repository;
    * step 5: change the `config/manager/all_in_one.yaml` and replace the image of the kubedl deployment to `$DOCKERID/kubedl:test`

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
    
    * step 6: `kubectl delete deployment kubedl-controller-manager -n kubedl` to remove the old deployment if any;
    * step 7: `kubectl apply -f config/manager/all_in_one.yaml` to install the new deployment with the customized operator image;

You can now perform manual tests and use `kubectl logs kubedl-controller-manager-0 -n kubedl-system` to check controller logs for debugging.