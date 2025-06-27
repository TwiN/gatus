# Generate ca-certificates file using sample certificates
FROM alpine as builder
RUN apk --update add ca-certificates
COPY certs/server/ca.crt /usr/local/share/ca-certificates/docker-compose-mtls.crt
RUN cat /usr/local/share/ca-certificates/docker-compose-mtls.crt >> /etc/ssl/certs/ca-certificates.crt

# Add CA cert to gatus
FROM twinproduction/gatus:latest
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
