# Build Instructions

## Correct Build Commands

The `go.mod` file is located in the `server` directory, so all Go commands must be run from there.

### Download Dependencies
```bash
cd server
go mod download
```

### Build All Packages
```bash
cd server
go build ./...
```

### Build Specific Components
```bash
cd server

# Build API server
go build -o bin/api ./cmd/api

# Build workers
go build -o bin/build-worker ./cmd/build-worker
go build -o bin/deploy-worker ./cmd/deploy-worker
go build -o bin/cleanup-worker ./cmd/cleanup-worker
```

### Clean Build (if you get caching issues)
```bash
cd server
go clean -cache
go mod download
go build ./...
```

## Common Issues

### Error: "no modules specified"
- **Cause**: Running `go mod download` from the wrong directory
- **Solution**: Make sure you're in the `server` directory

### Error: "undefined: hasDockerCompose"
- **Cause**: This variable doesn't exist in the current codebase
- **Solution**: 
  1. Clean the build cache: `go clean -cache`
  2. Rebuild: `go build ./...`
  3. If the error persists, check if you have uncommitted changes or a different version of the file

### Build Succeeds Locally but Fails in CI/CD
- Check if CI/CD is running from the correct directory
- Ensure `go.mod` and `go.sum` are committed
- Try `go mod tidy` to ensure dependencies are correct

