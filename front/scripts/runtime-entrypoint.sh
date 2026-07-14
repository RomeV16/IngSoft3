#!/usr/bin/env bash
# runtime-entrypoint.sh — INYECCIÓN DE VARIABLES EN RUNTIME.
#
# Problema que resuelve: NEXT_PUBLIC_API_URL en Next.js se fija en BUILD
# time, pero la misma imagen Docker se usa en DEV y PROD con backends
# distintos. Este script corre COMO ENTRYPOINT al arrancar el container
# (antes que Next.js) y "puentea" la variable de entorno al navegador:
#
#   Railway (env var) → este script → public/runtime-env.js → window.__ENV__
#
# El frontend carga runtime-env.js con un <script> y resolveApiBase() en
# api.ts lee window.__ENV__.NEXT_PUBLIC_API_URL con prioridad.

set -euo pipefail

# Prioridad: NEXT_PUBLIC_API_URL > API_URL > localhost (desarrollo)
API_URL="${NEXT_PUBLIC_API_URL:-${API_URL:-http://localhost:8080}}"

# Genera el JS que define window.__ENV__ en el navegador.
# Se escribe en /app/public para que Next.js lo sirva como archivo estático.
cat <<EOF >/app/public/runtime-env.js
window.__ENV__ = window.__ENV__ || {};
window.__ENV__.NEXT_PUBLIC_API_URL = "${API_URL}";
EOF

export NEXT_PUBLIC_API_URL="${API_URL}"

# exec reemplaza este proceso por el comando real del container (next start),
# así Next.js queda como PID 1 y recibe las señales de shutdown de Railway.
exec "$@"
