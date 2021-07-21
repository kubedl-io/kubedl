# KubeDL Console

## Prerequisites

- NodeJS > 10
- Go > 1.12

## Development

#### Build Console Backend Server
```bash
# kubedl/console/
go build -mod=mod -o backend-server ./backend/cmd/backend-server/main.go
```

#### Run local Console Backend Server

1. Create namespace `kubedl-system` in your k8s and make sure you have permission to create object.
2. Run backend server with disabled authentication mode
    ```bash
    ./backend-server
    ```
#### Optional
1. base images: You Can input some image names as base images when creating a job.
   Prepare the ConfigMap as below. These images will give you some choice when input job image.
    ``` yaml
    apiVersion: v1
     kind: ConfigMap
     metadata:
         namespace: kubedl-system
         name: kubedl-option-config
     data:
         images: '{
             "tf-cpu-images":[
               " here input your base image",
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
2. authorize: You can input some accounts into `users` in ConfigMap above, so that dashboard would check `uid` and `password` when login.
    ``` yaml
    apiVersion: v1
     kind: ConfigMap
     metadata:
         namespace: kubedl-system
         name: kubedl-option-config
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
   Start server in authorize mode.
    ```bash
    ./backend-server --auth-type=config
    ```
#### Run Console Frontend

```bash
cd frontend/
```

1. Install dependencies (optional)
    ```bash
    npm install
    ```
2. Run Console Frontend Dev Server
    ```bash
    npm run build
    ```
3. Move dist dir to `kubedl/console/`. Make sure that `dist` from frontend and `backend-server` from backend are in same path.
    ```bash
    cp -r dist ../
   
   
      |-- dist
      |-- .backend-server
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