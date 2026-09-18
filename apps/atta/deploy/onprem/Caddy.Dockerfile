FROM caddy:2-alpine
COPY apps/atta/deploy/onprem/Caddyfile /etc/caddy/Caddyfile
COPY apps/atta/pwa/dist/pwa/browser /srv
