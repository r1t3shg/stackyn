# Task Queue System

This package implements asynchronous task processing using Redis and Asynq.

## Architecture

- **Redis**: Task queue backend
- **Asynq**: Task queue library for Go
- **PostgreSQL**: Task state persistence
- **Separate Worker Binaries**: Each worker type runs as a separate process

## Task Types

1. **build_task**: Builds Docker images from source code
2. **deploy_task**: Deploys containers and configures networking
3. **cleanup_task**: Cleans up containers, images, and resources

## Features

### Retries
- Build tasks: 3 retries
- Deploy tasks: 3 retries
- Cleanup tasks: 2 retries

### Exponential Backoff
- Configured at server level
- Formula: 2^n seconds, max 30 seconds
- Applied automatically on retries

### Task Priority
Tasks are routed to queues based on priority:
- **Priority 0-3**: `low` queue
- **Priority 4-7**: `default` queue
- **Priority 8-10**: `critical` queue

Queues are processed with strict priority (critical first).

### Dead-Letter Queue
- Tasks that exceed max retries are automatically moved to dead-letter queue
- Dead-letter queue is monitored every minute
- Tasks in dead-letter queue are logged for manual review

### Task State Persistence
Task states are persisted in PostgreSQL `task_states` table:
- `pending`: Task enqueued
- `processing`: Task being processed
- `retrying`: Task failed, will retry
- `completed`: Task completed successfully
- `failed`: Task failed permanently

## Usage

### Enqueueing Tasks

```go
import "stackyn/server/internal/tasks"

// Create task client
client := tasks.NewTaskClient(redisAddr, logger)
defer client.Close()

// Enqueue build task
payload := tasks.BuildTaskPayload{
    AppID:      "app-123",
    BuildJobID: "build-456",
    RepoURL:    "https://github.com/user/repo",
    Branch:     "main",
}
taskInfo, err := client.EnqueueBuildTask(payload, 8) // Priority 8 = critical
```

### Running Workers

Each worker runs as a separate binary:

```bash
# Build worker
./bin/build-worker.exe

# Deploy worker
./bin/deploy-worker.exe

# Cleanup worker
./bin/cleanup-worker.exe
```

All workers process tasks from the same Redis queue but can be scaled independently.

## Configuration

Set Redis connection in environment variables:
- `REDIS_HOST`: Redis host (default: localhost)
- `REDIS_PORT`: Redis port (default: 6379)
- `REDIS_PASSWORD`: Redis password (optional)
- `REDIS_DB`: Redis database number (default: 0)

Or set `REDIS_ADDR` directly: `REDIS_ADDR=localhost:6379`

## Monitoring

Use Asynq's built-in monitoring tools or the dead-letter queue monitoring that logs queue statistics every minute.

## Database Schema

Task states are stored in `task_states` table with:
- Task ID (unique)
- Task type
- Queue name
- Payload (JSONB)
- Status
- Retry count
- Error messages
- Timestamps

