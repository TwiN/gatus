#!/usr/bin/env bash
set -euo pipefail

REALM="EXAMPLE.COM"
MASTER_PASSWORD="masterpassword"
GATUS_PRINCIPAL="gatus@EXAMPLE.COM"
HTTP_PRINCIPAL="HTTP/apache.example.com@EXAMPLE.COM"

mkdir -p /shared
chmod 755 /shared

if [ ! -f /var/lib/krb5kdc/principal ]; then
  echo "[KDC] Initializing Kerberos database..."
  kdb5_util create -s -r "${REALM}" -P "${MASTER_PASSWORD}"

  echo "[KDC] Creating principals..."
  kadmin.local -q "addprinc -pw gatus-password ${GATUS_PRINCIPAL}"
  kadmin.local -q "addprinc -randkey ${HTTP_PRINCIPAL}"

  echo "[KDC] Exporting keytabs..."
  kadmin.local -q "ktadd -k /shared/gatus.keytab ${GATUS_PRINCIPAL}"
  kadmin.local -q "ktadd -k /shared/apache-http.keytab ${HTTP_PRINCIPAL}"

  chmod 644 /shared/gatus.keytab
  chmod 644 /shared/apache-http.keytab

  echo "[KDC] Keytabs created:"
  klist -kte /shared/gatus.keytab
  klist -kte /shared/apache-http.keytab
fi

echo "[KDC] Starting krb5kdc..."
exec krb5kdc -n