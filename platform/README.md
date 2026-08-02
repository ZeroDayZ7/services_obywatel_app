# Platform Management

Automation target for managing dependencies, Go workspaces, and Docker services across microservices and shared packages.

## Directory Structure

- **`services/`** - Contains microservices (`gateway`, `auth-service`, `citizen-docs`, `notification-service`, `audit-service`, `version-service`).
- **`pkg/`** - Shared Go packages and common libraries.

## Available Commands

| Command             | Description                                                                                                                     |
| :------------------ | :------------------------------------------------------------------------------------------------------------------------------ |
| `make updatedeps`   | Upgrades external dependencies (`go get -u ./...`) and runs `go mod tidy` in all microservices and `pkg`, then syncs `go.work`. |
| `make deps-sync`    | Synchronizes the Go workspace (`go work sync`) and runs `go mod tidy` across modules.                                           |
| `make deps-upgrade` | Safely upgrades external libraries without affecting local `pkg` replace directives.                                            |
| `make deps-tidy`    | Runs `go mod tidy` in all services and the `pkg` directory.                                                                     |
| `make build`        | Builds Docker images for all configured services.                                                                               |
| `make up`           | Starts Docker Compose containers in detached mode.                                                                              |
| `make down`         | Stops Docker Compose containers.                                                                                                |
| `make update-all`   | Full execution pipeline: syncs deps, builds images, and restarts containers.                                                    |

## Usage

Run from the root directory:

```bash
make -C platform <command>

make -C platform updatedeps
```
