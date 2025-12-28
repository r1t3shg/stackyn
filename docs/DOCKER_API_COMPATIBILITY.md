# Docker API Version Compatibility Fix

## Problem

Stackyn uses Cloud Native Buildpacks (CNB) via pack CLI to build container images. Pack CLI v0.39.1 uses Docker client API version 1.42, but Docker Engine v29.0.0+ requires API version 1.44+.

**Error Message:**
```
ERROR: failed to initialize analyzer: getting previous image: Error response from daemon: client version 1.42 is too old. Minimum supported API version is 1.44, please upgrade your client to a newer version
```

## Solution: Configure Docker Daemon to Accept Older API Versions

Since pack CLI's Docker client version is bundled with the binary and cannot be updated independently, we need to configure the Docker daemon to accept older API versions.

### Option 1: Docker Daemon Configuration (Recommended)

Edit the Docker daemon configuration file:

**On Linux:**
```bash
sudo nano /etc/docker/daemon.json
```

**Add the following configuration:**
```json
{
  "min-api-version": "1.24"
}
```

**If the file already exists, merge the `min-api-version` setting with existing configuration.**

**Restart Docker daemon:**
```bash
sudo systemctl restart docker
```

### Option 2: Environment Variable (Alternative)

If you can't edit `/etc/docker/daemon.json`, you can set an environment variable:

**For systemd services:**
```bash
sudo systemctl edit docker.service
```

**Add:**
```ini
[Service]
Environment="DOCKER_MIN_API_VERSION=1.24"
```

**Reload and restart:**
```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### Option 3: Docker Compose Override (For Docker Compose setups)

If you're using Docker Compose, you can override the Docker daemon configuration by setting environment variables in your `docker-compose.yml` or via the host's Docker daemon configuration.

## Verification

After applying the configuration, verify it's working:

```bash
# Check Docker API version
docker version

# Test pack CLI build (should work now)
pack build test-image --builder paketobuildpacks/builder:base --path /path/to/app
```

## Long-term Solution

This is a temporary workaround. The proper solution is:

1. **Update pack CLI** when a newer version with Docker API 1.44+ support is released
   - Check: https://github.com/buildpacks/pack/releases
   - Update `PACK_VERSION` in `backend/Dockerfile`
   - Rebuild: `docker compose build worker`

2. **Monitor pack CLI releases** for Docker API 1.44+ compatibility

## Important Notes

- This configuration allows older Docker clients (API 1.24+) to communicate with Docker Engine v29+
- This is safe for development and staging environments
- For production, consider waiting for pack CLI to be updated, or using Docker Engine v28.x temporarily
- The `min-api-version` setting doesn't affect security, it only controls API version compatibility

## Related Issues

- Docker Engine v29 Release Notes: https://www.docker.com/blog/docker-engine-version-29/
- Pack CLI GitHub: https://github.com/buildpacks/pack

