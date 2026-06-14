#!/bin/bash
set -e

CERT_DIR="certs"
mkdir -p "$CERT_DIR"

echo "🔐 Генерация self-signed сертификатов для Support-Radar..."

openssl req -x509 -newkey rsa:4096 \
    -keyout "$CERT_DIR/server.key" \
    -out "$CERT_DIR/server.crt" \
    -days 365 \
    -nodes \
    -subj "/CN=support-radar/O=Support-Radar/C=RU" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1"

openssl req -x509 -newkey rsa:4096 \
    -keyout "$CERT_DIR/client.key" \
    -out "$CERT_DIR/client.crt" \
    -days 365 \
    -nodes \
    -subj "/CN=agent/O=Support-Radar/C=RU"

echo "✅ Сертификаты созданы в $CERT_DIR/"