# KubeDL-Pro Console

## Prerequisites

- NodeJS > 10
- Go > 1.12

## Development

#### Build Console Backend Server
```bash
go build -mod=mod -o backend-server github.com/alibaba/kubedl/console/backend/cmd/backend-server
```

#### Run local Console Backend Server

1. Prepare a `kubeconfig` file which defines k8s development environment.
2. Set `KUBECONFIG` environment variable.
```bash
export KUBECONFIG={/path-to-kubeconfig-file} 
```
3. Run backend server with disabled authentication mode

```bash
./backend-server --config-name kubedl-config
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
npm run start
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