FROM golang:1.23-alpine as builder
WORKDIR /go/src/github.com/adamdecaf/deadcheck
RUN apk add -U git make
RUN adduser -D -g '' --shell /bin/false runner
COPY . .
RUN make build
USER runner

FROM alpine:3
LABEL maintainer="Adam Shannon <adamkshannon@gmail.com>"
RUN apk add --no-cache tzdata

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/src/github.com/adamdecaf/deadcheck/bin/deadcheck /bin/deadcheck
COPY --from=builder /etc/passwd /etc/passwd

USER runner
EXPOSE 8080
EXPOSE 9090
ENTRYPOINT ["/bin/deadcheck"]
