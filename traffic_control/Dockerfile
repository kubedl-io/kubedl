FROM golang:1.16 AS builder
RUN GO111MODULE=on go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
RUN xcaddy version
COPY . /src
WORKDIR /src
RUN xcaddy build --with github.com/kubedl/istio_less_ingress_controller=/src

FROM golang:1.16
COPY . /src
COPY --from=builder /src/caddy /src
WORKDIR /src
CMD ["./caddy", "run"]