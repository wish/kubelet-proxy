FROM       golang:alpine as builder

COPY . /go/src/github.com/wish/kubelet-proxy
RUN cd /go/src/github.com/wish/kubelet-proxy/cmd/kubelet-proxy && CGO_ENABLED=0 go build

FROM golang:alpine

COPY --from=builder /go/src/github.com/wish/kubelet-proxy/cmd/kubelet-proxy/kubelet-proxy /bin/kubelet-proxy
