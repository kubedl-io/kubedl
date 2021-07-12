# KubeDL Console

## Prerequisites

- NodeJS > 10
- Go > 1.12

## Development

#### Build Console Backend Server
```bash
go build -mod=mod -o backend-server github.com/alibaba/kubedl/console/backend/cmd/backend-server
```

#### Run local Console Backend Server

1. Create namespace `kubedl-system` in your k8s and make sure you have permission to create object.
2. Prepare a ConfigMap as below. If not provide, kubedl will create default `kubedl-config` in namespace `kubedl-system`.
    ```
    apiVersion: v1
    kind: ConfigMap
    metadata:
        namespace: kubedl-system
        name: kubedl-config
    data:
        commonConfig: '{
            "namespace":"kubedl-system",
            "TFCpuImages":"",
            "TFGpuImages":"",
            "PytouchGpuImages":""
        }'
        users: '{
            "admin":{
            "uid":"admin",
            "login_name":"admin",
            "password":"123456"
            }
        }'
    ```
    ConfigMap requires default namespace, images and dashboard login accounts.

3. Run backend server with disabled authentication mode
    ```bash
    ./backend-server --config-name kubedl-config
    ```
    You can input some accounts into `users` in above ConfigMap, so that dashboard would check authorize when login.
    ```bash
    ./backend-server --auth-type=config
    ```
#### Run Console Frontend

```bash
cd console/frontend/
```

1. Install dependencies (optional)
    ```bash
    npm install
    ```
2. Run Console Frontend Dev Server
    ```bash
    npm run build
    ```
3. Move target dir to project dir.
    ```bash
    cp -r dist ../../
    ```
#### Optional: Start Console Frontend with Connection to other dev Backend-Server directly
If you are not able to run local console backend server, or other dev console backend server is already present, you could make frontend dev server to proxy API requests to other dev backend server directly.

1. Change Proxy Backend
Path: console/frontend/config/config.js
```javascript
  proxy: [
    {
      target: "http://localhost:9090",
      ...
    }
  ]

```
change the target to address <ip:port> of other present console backend server.


2. Run Console Frontend Dev Server
```bash
npm run start
```

## Tools

#### Editor Recommandation

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