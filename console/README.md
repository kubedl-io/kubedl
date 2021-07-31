
KubeDL dashboard consists of a frontend and a backend. Below documentation describes how to build and run them.

## Prerequisites

- NodeJS > 10
- Go > 1.12
## Deployment Guide

### Deploy the KubeDL Dashboard

```bash
kubectl apply -f console/dashboard.yaml
```
This will create a kubedl-dashboard `Deployment`, its `Service`, and a `ConfigMap` in the `kubedl-system` namespace.

The dashboard will list nodes. Hence, its service account requires the ``list node permission``.

### Access the Dashboard

You can access the dashboard by the ClusterIP or LoadBalancer IP or Ingress depending on your own usage.

For example, check the dashboard endpoint by inspecting the service object and you can find the access endpoint.

```bash
 kubectl describe service kubedl-dashboard-service -n kubedl-system
```

#### Access the Dashboard over SSH

If the dashboard is deployed on a remote machine that requires SSH to access. Run

```
ssh -L 9090:localhost:9090 30.30.30.30
```
This will send any browser connection to port 9090 on your local machine(i.e. your laptop), over ssh to the remote machine (30.30.30.30).
Once there, it will continue to localhost (the remote machine), port 9090.

Then, on the remote machine, run

```bash
 kubectl port-forward deployment/kubedl-dashboard -n kubedl-system 9090:9090
```

This will forward any connections to localhost:9090 (the remote machine you ssh to) to the kubedl-dashboard deployment in Kuberenetes at port 9090

In summary, the connection is flow like below:

Browser -> Local Machine (e.g. your laptop), port 9090 -> Remote Machine, port 9090 -> kubectl forward -> The running pod, port 9090

## Development Guide

### Build the KubeDL Dashboard Image

```
docker build . -t kubedl/dashboard:0.1.0 -f Dockerfile.dashboard
```

### Build Backend Server Binary
```bash
$ cd console/
$ go build -o backend-server ./backend/cmd/backend-server/main.go
```

### Run Backend Server Locally

1. Create a `kubedl-system` namespace in your Kubernetes if not existing, this is required to create system-level ConfigMaps.
2. Make sure the backend-server uses a `KUBECONFIG` that has permission to create ConfigMap.
2. Run backend server with no authentication (default mode).
    ```bash
    export KUBECONFIG=<path/to/your/kubeconfig> && ./backend-server
    ```

#### Optional Settings
1. Default Training Container Images

    You can set the default container images for submitting the training jobs through dashboard by creating a `ConfigMap`
    named `kubedl-dashboard-config` in `kubedl-system` namespace as below:
    ``` yaml
     apiVersion: v1
     kind: ConfigMap
     metadata:
         namespace: kubedl-system
         name: kubedl-dashboard-config #
     data:
         images: '{
             "tf-cpu-images":[
               "here set your default container image",
               ...
             ],
            "tf-gpu-images":[
               ...
            ],
            "pytorch-gpu-images":[
               ...
            ]
         }'
    ```

2. Authentication

    By default, the backend-server has no authentication.
    Optionally, you can enable authentication using ConfigMap. That is, use `username` and `password` defined in ConfigMap and to login.

    The backend-server needs to start as `./backend-server --authentication-mode=config`.
    For example, create a ConfigMap like below:

    ``` yaml
    apiVersion: v1
     kind: ConfigMap
     metadata:
         namespace: kubedl-system
         name: kubedl-dashboard-config
     data:
        images:
               ...
        users: '[
            {
            "username":"admin",
            "password":"123456"
            }
        ]'
    ```
    When login to the frontend, use `admin` for username and `123456` for password to login.

### Run Frontend

1. Go to the frontend root dir.
    ```bash
    cd frontend/
    ```

2. Install dependencies
    ```bash
    npm install
    ```
3. Build Frontend Server
    ```bash
    npm run build
    ```
4. Start Frontend Server

    ```bash
    npm start
    ```

#### Optional: Set backend server address

1. Use below config to set the backend-server address.

    Path: console/frontend/config/config.js
    ```javascript
      proxy: [
        {
          target: "http://localhost:9090",
          ...
        }
      ]
    ```

Change the target to your own backend server address. By default, it is `localhost:9090`.



### Editor Tool Recommandation

VSCode + ESlint(Plugin)

VSCode Configuration:
```javascript
{
    "eslint.run": "onSave",
    "eslint.format.enable": true,
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
        "source.fixAll.eslint": true
    }
}
```

### Check code style

```bash
npm run lint
```

You can also use script to auto fix some lint error:

```bash
npm run lint:fix
```

### Test code

```bash
npm test
```
