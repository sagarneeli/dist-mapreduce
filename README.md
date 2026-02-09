# Distributed MapReduce System in Go

## Overview
This project is a modern implementation of a Distributed MapReduce system, written in **Go**. It serves as a comprehensive modernization of a legacy Java/Hadoop project, re-architected to showcase **Distributed Systems** concepts, **backend engineering** skills, and **cloud-native** readiness (Docker/Containerization).

## Architecture
The system follows a classic **Master-Worker** architecture:

- **Coordinator (Master)**: 
  - Manages the job lifecycle.
  - Assigns Map and Reduce tasks to available workers.
  - Monitors worker health and task progress (basic fault tolerance).
  
- **Workers**: 
  - Stateless processes that register with the Coordinator via RPC.
  - Execute Map functions (Word Count) on input shards.
  - Execute Reduce functions (Summation) on intermediate data.

### Key Technologies
- **Go**: Chosen for strong concurrency primitives (Channels/Goroutines) and performance.
- **RPC**: Custom RPC scheduler for low-latency task coordination.
- **Docker & Docker Compose**: Fully containerized environment for reproducible deployments.
- **Makefile**: Automation for build and test workflows.

## Getting Started

### Prerequisites
- **Go** 1.25+ (for local development)
- **Docker** & **Docker Compose**
- **Make**

### 🚀 Quick Start (Docker)
The easiest way to run the system is using Docker Compose. This spins up the Coordinator and multiple Workers.

```bash
# 1. Start the cluster
make docker-up

# 2. View the output (in another terminal)
# The output files are generated in ./data/
cat data/mr-out-*
```

### 🛠️ Local Development
To run the components manually on your host machine:

1. **Build the projects**:
   ```bash
   make build
   ```

2. **Generate/Prepare Data**:
   Ensure you have input files in `data/input/`.
   ```bash
   mkdir -p data/input
   echo "Hello distributed world" > data/input/test1.txt
   ```

3. **Start Coordinator**:
   ```bash
   # Usage: ./bin/coordinator <input_files>
   ./bin/coordinator data/input/*.txt
   ```

4. **Start Workers** (Run in separate terminals):
   ```bash
   ./bin/worker
   ```


## Testing
The project includes comprehensive unit tests for both Coordinator and Worker components.

### Running Tests
To run all tests:
```bash
go test -v ./...
```

### Test Coverage
- **Worker**: Validates Map/Reduce logic, word splitting, and hashing.
- **Coordinator**: Validates task assignment, worker registration, and job completion logic.

### Code Quality
The project is configured with automatic linting and formatting.

- **Format Code**:
  ```bash
  make fmt
  ```

- **Run Linters**:
  ```bash
  make lint
  ```
  *(Requires `golangci-lint` installed locally, or falls back to `go vet`)*

- **CI/CD**:
  GitHub Actions are configured in `.github/workflows/ci.yml` to automatically:
  - Run **Linting** (`golangci-lint`)
  - Run **Unit Tests** with Race Detection
  - **Build** Binaries for Coordinator and Worker
  - **Build** Docker Images to ensure valid configuration

## Project Structure

```
.
├── cmd/                # Entrypoints
│   ├── coordinator/    # Master service main
│   └── worker/         # Worker service main
├── internal/
│   ├── coordinator/    # Task scheduling and state logic
│   ├── worker/         # Map/Reduce implementation
│   └── common/         # RPC definitions and shared types
├── data/               # Mounted directory for Input/Output
├── Dockerfile.*        # Container definitions
├── docker-compose.yml  # Orchestration
└── Makefile            # Build automation
```

## Legacy Comparison

| Feature | Legacy Code (2016) | Modern Rewrite (2026) |
|---------|-------------------|-----------------------|
| **Language** | Java | Go |
| **Framework** | Hadoop MapReduce | Custom Distributed System |
| **Infrastructure** | Manual / VM | Docker Containers |
| **Architecture** | Monolithic Job | Microservices / RPC |

## API Reference
The Coordinator exposes a REST API on port `8080`.

- **Submit Job**
  ```bash
  curl -X POST http://localhost:8080/jobs -d '{"files": ["/app/data/input/test1.txt"], "nReduce": 10}'
  ```

- **Check Job Status**
  ```bash
  curl http://localhost:8080/jobs/0
  ```

- **Health Check**
  ```bash
  curl http://localhost:8080/health
  ```

## Future Improvements
- [ ] **Advanced Fault Tolerance**: Handle worker crashes by re-assigning in-progress tasks after a timeout.
- [ ] **Dynamic Scaling**: Integrate with Kubernetes to auto-scale workers based on load.
- [ ] **Universal Serialization**: Replace `gob`/JSON with Protobuf/gRPC for language-agnostic workers.
