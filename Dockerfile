FROM golang:1.23.1 as builder
LABEL author="gitlayzer"
LABEL email="gduxintian@gmail.com"
ENV LANG C.UTF-8
WORKDIR /tsunami
COPY . .
ENV GO111MODULE on
ENV GOPROXY https://goproxy.cn
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o tsunami ./cmd/pod
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o cni-tsunami ./cmd/cni

FROM alpine:3.20
LABEL author="gitlayzer"
LABEL email="gduxintian@gmail.com"
ENV LANG C.UTF-8
COPY --from=builder /tsunami/tsunami /
COPY --from=builder /tsunami/cni-tsunami /

CMD ["/tsunami"]
