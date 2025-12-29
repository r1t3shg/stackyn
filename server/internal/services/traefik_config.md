# Traefik Configuration via Docker Labels

This document describes the Traefik configuration implemented via Docker labels.

## Requirements

Traefik must be configured with:
- Entrypoints: `web` (HTTP on port 80) and `websecure` (HTTPS on port 443)
- TLS certificate resolver: `letsencrypt` (or another ACME provider)
- Docker provider enabled
- Network: `traefik` (or configured network name)

## Label Configuration

Each deployed container receives the following Traefik labels:

### Basic Configuration
- `traefik.enable=true` - Enables Traefik for this container
- `traefik.docker.network=traefik` - Specifies the Docker network

### HTTP Router (Redirects to HTTPS)
- `traefik.http.routers.{routerName}-http.rule=Host(\`{subdomain}\`)` - Matches HTTP requests
- `traefik.http.routers.{routerName}-http.entrypoints=web` - Listens on HTTP entrypoint
- `traefik.http.routers.{routerName}-http.middlewares={middlewareName}` - Applies redirect middleware

### HTTPS Router (Main Router)
- `traefik.http.routers.{routerName}.rule=Host(\`{subdomain}\`)` - Matches HTTPS requests
- `traefik.http.routers.{routerName}.entrypoints=websecure` - Listens on HTTPS entrypoint
- `traefik.http.routers.{routerName}.tls=true` - Enables TLS
- `traefik.http.routers.{routerName}.tls.certresolver=letsencrypt` - Uses Let's Encrypt for certificates

### Service Configuration
- `traefik.http.services.{serviceName}.loadbalancer.server.port={port}` - Container port
- `traefik.http.services.{serviceName}.loadbalancer.healthcheck.path=/` - Health check path
- `traefik.http.services.{serviceName}.loadbalancer.healthcheck.interval=10s` - Check interval
- `traefik.http.services.{serviceName}.loadbalancer.healthcheck.timeout=3s` - Check timeout

### Redirect Middleware
- `traefik.http.middlewares.{middlewareName}.redirectscheme.scheme=https` - Redirects to HTTPS
- `traefik.http.middlewares.{middlewareName}.redirectscheme.permanent=true` - Permanent redirect (301)

### App Labels
- `app.id={appID}` - Application identifier
- `app.subdomain={subdomain}` - Subdomain for this app

## Example

For an app with:
- App ID: `app-123`
- Subdomain: `myapp.example.com`
- Port: `8080`

The generated labels would be:
```
traefik.enable=true
traefik.docker.network=traefik
traefik.http.routers.app-app-123-http.rule=Host(`myapp.example.com`)
traefik.http.routers.app-app-123-http.entrypoints=web
traefik.http.routers.app-app-123-http.middlewares=app-app-123-redirect
traefik.http.routers.app-app-123.rule=Host(`myapp.example.com`)
traefik.http.routers.app-app-123.entrypoints=websecure
traefik.http.routers.app-app-123.tls=true
traefik.http.routers.app-app-123.tls.certresolver=letsencrypt
traefik.http.services.app-app-123.loadbalancer.server.port=8080
traefik.http.services.app-app-123.loadbalancer.healthcheck.path=/
traefik.http.services.app-app-123.loadbalancer.healthcheck.interval=10s
traefik.http.services.app-app-123.loadbalancer.healthcheck.timeout=3s
traefik.http.middlewares.app-app-123-redirect.redirectscheme.scheme=https
traefik.http.middlewares.app-app-123-redirect.redirectscheme.permanent=true
app.id=app-123
app.subdomain=myapp.example.com
```

## Traefik Configuration Requirements

Traefik must be configured with the following static configuration:

```yaml
entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
          permanent: true
  websecure:
    address: ":443"

certificatesResolvers:
  letsencrypt:
    acme:
      email: your-email@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: traefik
```

## Notes

- All routing is done via Docker labels - no manual configuration required
- HTTPS is automatically enabled with Let's Encrypt certificates
- Health checks are configured at both Traefik and Docker levels
- Subdomains are automatically configured per app
- HTTP requests are automatically redirected to HTTPS

