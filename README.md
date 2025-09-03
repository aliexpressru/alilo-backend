# Alilo Backend
<img src="./images/logo.png" alt="Logo" 
     style="max-width: 300px; height: auto; display: block; margin-left: auto; margin-right: auto;">

**AliLo** Backend is the central service of the Alilo (**Ali**express **Lo**ad) ecosystem. It integrates the frontend, database (metadata), MinIO (artifact storage), and agent services to provide comprehensive management of load scripts and orchestrate load test execution.

## Scheme
<img src="./images/scheme.png" alt="Logo" 
     style="max-width: 400px; height: auto; display: block; margin-left: auto; margin-right: auto;">

The Alilo Backend is the core service that powers the load-testing platform. It provides the APIs and business logic for all operations, built upon a structured data model.

### Key Backend Capabilities:

- Provides RESTful APIs for frontend interaction.
- Manages the entire lifecycle of load tests.
- Orchestrates test execution with real-time control over load intensity.
- Parses and processes load test scenarios, including those imported from cURL commands.

### Demo
To demonstrate the functionality, clone the repository and launch all containers.

```console
git clone https://github.com/aliexpressru/alilo-backend.git
docker compose up -d
```

**Access the following services:**

- **Frontend**: http://localhost
- **Backend Swagger**: http://localhost:8084/swagger#/
- **MinIO UI**: http://localhost:9001/login (credentials: minioadmin/minioadmin)
- **Agent API**: http://localhost:8888/api/ (see full [endpoint list](https://github.com/aliexpressru/alilo-agent?tab=readme-ov-file#agent-endpoints))

### Core Domain Models:

- Project: The top-level domain or product grouping.
- Scenario: A service or functional group containing multiple scripts.
- Script: An entity representing a single API endpoint or test scenario.
- Run: A model managing the state, metrics, and real-time control of an ongoing test execution.

## Dependencies
The Alilo Backend service relies on the following external components to operate:

- Minio: Used as an S3-compatible object storage service for reliable storage and management of script files, configuration profiles, and test artifacts.

- Postgres: Serves as the primary relational SQL database for storing all structured data, including project metadata, scenarios, scripts, test run history, and results.

- Alilo-agent: A lightweight client deployed on load-generating machines. The backend orchestrates tests by sending commands to these agents (start/stop), which are responsible for the actual execution of load scripts and generating traffic against target systems.

## Development Environment Setup
This section outlines the steps required to set up a local development environment for the Alilo Backend project. The service relies on several external dependencies: PostgreSQL (as the primary database), MinIO (as an S3-compatible object storage for files), and Goose (a database migration tool).

### 1. Configure Go Environment
First, ensure your Go environment variables are correctly set. This configures your `GOPATH` and adds the Go binary directory to your `PATH`.

```console
export GOPATH="$HOME/go"
export PATH="$(pwd)/bin:$PATH"
```

### 2. Install GNU Make
We are using GNU Make to build the project. You can install it using the following commands:

```console
# Use for macOS
brew install make

# Use for Linux
sudo apt-get install make

# Use for Windows
choco install make
```

### 3. Install Project Dependencies

Download and sync all the Go module dependencies required by the project. The `vendor` command creates a local copy of the dependencies for reproducible builds.

```console
go mod tidy
go mod vendor
```

### 4. Install Required System Utilities

The project requires several development tools that are automatically installed and managed by our Makefile. These tools handle code generation, database operations, code quality, and build processes.

#### Development Tools Overview:

**Protobuf & gRPC Tools:**
- **`protoc-gen-go`** - Generates Go code from Protocol Buffer definitions
- **`protoc-gen-grpc-gateway`** - Creates HTTP/gRPC gateway for REST API endpoints
- **`protoc-gen-openapiv2`** - Generates OpenAPI/Swagger documentation from Protobuf
- **`protoc-gen-go-grpc`** - Generates gRPC Go code for client/server communication

**Code Quality & Linting:**
- **`golangci-lint`** - Comprehensive Go linter that runs multiple linters in parallel
- **`buf`** - Modern Protobuf toolkit for linting, formatting, and managing Protobuf files

**Database Tools:**
- **`sqlboiler`** - Generates type-safe Go code for database operations
- **`sqlboiler-psql`** - PostgreSQL driver for SQLBoiler
- **`goose`** - Database migration tool for managing schema changes

**Build & Development:**
- **`modtools`** - Custom tool for copying non-Go dependencies into vendor directory

#### Installation Commands:

```console
# Install all required tools to local bin/ directory
make install-tools

# Generate Protobuf code and database models
make generate
```

### 5. Run Infrastructure with Docker Compose

The core dependencies (PostgreSQL and MinIO) can be run locally using Docker Compose. The configuration below:

- Creates a PostgreSQL instance with default credentials
- Creates a MinIO object storage server and automatically initializes a public bucket named test-data via the init-minio service
- Creates a dedicated Docker network (alilo-network) for inter-container communication

#### Create and run the stack:

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres_db:
    image: postgres:latest
    container_name: postgres_db
    restart: always
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=mysecretpassword
    ports:
      - '5432:5432'
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: [ "CMD-SHELL", "pg_isready -U postgres" ]
      interval: 5s
      timeout: 5s
      retries: 5
    networks:
      - alilo-network

  minio:
    container_name: minio
    image: quay.io/minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    restart: unless-stopped
    networks:
      - alilo-network

  init-minio:
    container_name: init-minio
    image: quay.io/minio/mc:RELEASE.2025-03-12T17-29-24Z
    depends_on:
      - minio
    restart: on-failure
    entrypoint: >
      /bin/sh -c "
      sleep 5;
      /usr/bin/mc alias set myminio http://minio:9000 minioadmin minioadmin;
      /usr/bin/mc mb myminio/test-data/test;
      /usr/bin/mc mb anonymous get myminio/test-data;
      /usr/bin/mc anonymous set public myminio/test-data;
      exit 0;
      "
    networks:
      - alilo-network

volumes:
  postgres_data:
    driver: local
  minio_data:
    driver: local

networks:
  alilo-network:
    driver: bridge
```

#### Start the services:

```console
docker-compose up -d
```

### 6. Configure Application Environment
Create or edit the `.env` file to provide the application with the correct connection strings for the locally running services. Note the use of `postgres_db` and `minio` as hostnames, which are resolvable within the Docker network.

```conf
# .env
PG_DSN=postgres://postgres:mysecretpassword@localhost:5432/postgres?sslmode=disable
MINIO_ENDPOINT=localhost:9000
...
```

### 7. Apply Database Migrations
After PostgreSQL is running, apply the database schema migrations using the `goose` tool. This command will execute all migration files located in `db/migrations`.

```console
make migrate-up
```

### 8. Build and Run the Application
Once the infrastructure is running and the migrations are applied, you can start the backend application.

```console
make run
```

### 9. Verify the Setup
To confirm that all services are interacting correctly, you can send a test API request. For example, use the following `curl` command to test the project creation endpoint.

```console
curl --location --request POST 'http://localhost:8084/v1/projects'
```
