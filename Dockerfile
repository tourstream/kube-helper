FROM golang:1.8 as builder
WORKDIR /go/src/kube-helper/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kube-helper .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/kube-helper/kube-helper /
ENTRYPOINT ["/kube-helper"]