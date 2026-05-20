FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    apache2 \
    libapache2-mod-auth-gssapi \
    krb5-user \
    netcat-openbsd \
    && rm -rf /var/lib/apt/lists/*

COPY gssapi.conf /etc/apache2/sites-available/000-default.conf

RUN a2enmod auth_gssapi headers \
    && mkdir -p /var/www/html/secure \
    && echo "public ok" > /var/www/html/index.html \
    && echo "kerberos ok" > /var/www/html/secure/index.html

EXPOSE 80

CMD ["/bin/bash", "-c", "set -e; while [ ! -s /shared/apache-http.keytab ]; do echo 'Waiting for non-empty Apache keytab...'; sleep 1; done; cp /shared/apache-http.keytab /etc/apache2/apache-http.keytab; chown www-data:www-data /etc/apache2/apache-http.keytab; chmod 600 /etc/apache2/apache-http.keytab; klist -kte /etc/apache2/apache-http.keytab; apachectl -D FOREGROUND"]