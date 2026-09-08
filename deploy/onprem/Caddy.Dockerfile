FROM caddy:2-alpine
COPY deploy/onprem/Caddyfile /etc/caddy/Caddyfile
COPY apps/pwa/dist/pwa/browser /srv
