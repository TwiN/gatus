FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    krb5-kdc \
    krb5-admin-server \
    krb5-user \
    dnsutils \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

COPY krb5.conf /etc/krb5.conf
COPY init-kdc.sh /init-kdc.sh

RUN chmod +x /init-kdc.sh

EXPOSE 88/tcp 88/udp

CMD ["/init-kdc.sh"]