FROM --platform=$BUILDPLATFORM golang:1.14

ARG BUILDPLATFORM
ARG TARGETARCH
ARG TARGETOS

ENV GO111MODULE=on
WORKDIR /go/src/github.com/wish/kubelet-proxy

# Cache dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . /go/src/github.com/wish/kubelet-proxy/

RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} GOOS=${TARGETOS} go build -o ./kubelet-proxy -a -installsuffix cgo ./cmd/kubelet-proxy

FROM alpine:3.11
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/src/github.com/wish/kubelet-proxy/kubelet-proxy /bin/kubelet-proxy
