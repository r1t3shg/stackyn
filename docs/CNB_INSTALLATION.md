# Cloud Native Buildpacks (CNB) Installation Guide

## Overview

Stackyn uses Cloud Native Buildpacks (CNB) to automatically build container images from source code. The **pack CLI** tool is required and is automatically installed in the Stackyn Docker container.

## Installation Status

✅ **Already installed!** - The pack CLI is automatically installed in the Stackyn worker Docker container via the `backend/Dockerfile`. No manual installation required on your VPS.

## How It Works

When you build the Stackyn Docker image, the Dockerfile automatically:
1. Downloads the pack CLI binary
2. Installs it to `/usr/local/bin/pack`
3. Verifies the installation

The worker container has pack CLI available when it runs.

## Verification

After deploying Stackyn, you can verify pack CLI is available:

### Option 1: Check Worker Logs

When the worker starts, you should see in the logs:
```
[BUILDPACKS] Checking for pack CLI at: pack
[BUILDPACKS] Pack CLI version: 0.39.1+git-...
[BUILDPACKS] Buildpacks builder initialized successfully
```

### Option 2: Check Inside Container

```bash
# Enter the worker container
docker exec -it stackyn-worker sh

# Check pack CLI version
pack version

# You should see output like:
# 0.39.1+git-...
```

## If You Need to Update pack CLI

If you need to update to a newer version of pack CLI, update the version in `backend/Dockerfile`:

```dockerfile
# Change this line in backend/Dockerfile:
RUN PACK_VERSION="v0.39.1" && \
    # ... rest of installation commands
```

Then rebuild the Docker image:
```bash
docker compose build worker
docker compose up -d worker
```

## Manual Installation (if running outside Docker)

If you're running Stackyn worker directly on the VPS (not in Docker), you need to install pack CLI manually:

### Step 1: Download pack CLI

For Linux x86_64:
```bash
wget https://github.com/buildpacks/pack/releases/latest/download/pack-v0.39.1-linux.tgz
tar -xzf pack-v0.39.1-linux.tgz
chmod +x pack
sudo mv pack /usr/local/bin/pack
```

For Linux ARM64:
```bash
wget https://github.com/buildpacks/pack/releases/latest/download/pack-v0.39.1-linux-arm64.tgz
tar -xzf pack-v0.39.1-linux-arm64.tgz
chmod +x pack
sudo mv pack /usr/local/bin/pack
```

### Step 2: Verify Installation

```bash
pack version
```

## Troubleshooting

### Issue: "pack CLI not found" in worker logs

**If using Docker:**
1. Rebuild the Docker image: `docker compose build worker`
2. Restart the worker: `docker compose restart worker`
3. Check logs: `docker compose logs worker`

**If running directly on VPS:**
1. Ensure pack is in PATH: `which pack`
2. Install pack CLI (see Manual Installation above)

### Issue: "pack build fails with permission denied"

**Solution:**
- Ensure Docker socket is accessible: `ls -la /var/run/docker.sock`
- Worker container should have Docker socket mounted (already configured in docker-compose.yml)
- If running outside Docker, ensure user has Docker access: `sudo usermod -aG docker $USER`

### Issue: "pack build fails to pull builder image"

**Solution:**
- Ensure Docker has internet access to pull builder images
- Builder image `paketobuildpacks/builder:base` is large (~1GB) - ensure sufficient disk space
- Check Docker registry access if behind a firewall

### Issue: "client version 1.42 is too old. Minimum supported API version is 1.44"

**Problem:** Docker Engine v29.0.0+ requires API version 1.44+, but pack CLI v0.39.1 uses Docker client API 1.42.

**Solutions:**

1. **Update pack CLI to latest version** (Recommended):
   - Update `PACK_VERSION` in `backend/Dockerfile` to the latest pack CLI release
   - Check latest version at: https://github.com/buildpacks/pack/releases
   - Rebuild Docker images: `docker compose build worker`

2. **Configure Docker daemon to accept older API versions** (Temporary workaround):
   - Edit Docker daemon configuration (usually `/etc/docker/daemon.json`):
     ```json
     {
       "min-api-version": "1.24"
     }
     ```
   - Or set environment variable: `DOCKER_MIN_API_VERSION=1.24`
   - Restart Docker daemon: `sudo systemctl restart docker`
   - **Note:** This is a temporary workaround and not recommended for production

3. **Downgrade Docker Engine** (Not recommended):
   - Use Docker Engine v28.x which supports older API versions
   - Not recommended due to security and feature limitations
- Builder images are cached locally after first pull

## Builder Image

Stackyn uses **Paketo Buildpacks base builder** (`paketobuildpacks/builder:base`) which supports:
- Node.js (npm, yarn, pnpm, TypeScript)
- Python (pip, poetry, pipenv)
- Java (Maven, Gradle)
- Go (standard Go modules)
- .NET Core
- PHP
- Ruby
- And more...

The builder image is automatically pulled on first use (~1GB download). Subsequent builds reuse the cached image.

## Additional Resources

- Pack CLI Documentation: https://buildpacks.io/docs/tools/pack/
- Pack Releases: https://github.com/buildpacks/pack/releases
- Paketo Buildpacks: https://paketo.io/
- Cloud Native Buildpacks: https://buildpacks.io/
