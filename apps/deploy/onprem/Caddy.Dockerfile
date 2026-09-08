FROM caddy:2-alpine
COPY apps/deploy/onprem/Caddyfile /etc/caddy/Caddyfile
COPY apps/pwa/dist/pwa/browser /srv
