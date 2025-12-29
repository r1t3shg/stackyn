# Backend v2

This is the fresh v2 backend system for Stackyn, built from the ground up with a modern architecture.

## Architecture Overview

### Async Architecture

The backend v2 is designed with an asynchronous, event-driven architecture. This enables:

- **Non-blocking operations**: Long-running tasks (builds, deployments) are processed asynchronously
- **Scalability**: The system can handle multiple concurrent operations without blocking
- **Resilience**: Failed operations can be retried without affecting the main API
- **Separation of concerns**: API handlers remain lightweight while heavy processing happens in background workers

The architecture uses message queues and event handlers to decouple API requests from background processing, ensuring responsive API responses even during intensive operations.

### Single VPS Deployment

This backend is optimized for deployment on a single VPS (Virtual Private Server). The design considerations include:

- **Resource efficiency**: All components run on a single server, minimizing infrastructure costs
- **Simplified deployment**: No complex distributed system setup required
- **Unified logging and monitoring**: All services share the same environment
- **Cost-effective**: Ideal for startups and small-to-medium scale applications

The system is designed to efficiently utilize available resources while maintaining performance and reliability within a single-server environment.

### Frontend Contract Preserved

The v2 backend maintains full compatibility with the existing frontend API contract. This means:

- **Zero frontend changes required**: All existing API endpoints and response formats are preserved
- **Seamless migration**: The frontend can switch to v2 backend without any modifications
- **API compatibility**: Request/response schemas remain identical to the legacy backend
- **Backward compatible**: Existing integrations continue to work without changes

This ensures a smooth transition to v2 without disrupting the user experience or requiring frontend development work.

## Getting Started

This is a fresh implementation. Development setup and deployment instructions will be added as the system is built out.

