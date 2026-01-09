# Obywatel Platform - Microservices Monorepo

A sophisticated microservices ecosystem built with **Go**, leveraging a shared-resource architecture (**pkg**) and centralized management via **Go Workspaces**. The platform is engineered for high scalability, data integrity, and comprehensive observability.

## 🛠 Tech Stack

- **Backend Framework:** Go with **Fiber** (High-performance, Express-inspired web framework).
- **Databases:** **PostgreSQL** utilizing **SQLC** (Type-safe SQL generator) for the Audit Service and **GORM** (ORM) for domain-driven services.
- **Caching & State:** **Redis** for session management, distributed locking, and high-speed data caching.
- **Log Management:** **Zap Logger** for high-performance structured JSON logging, integrated with **Lumberjack** for automated log rotation and retention.
- **Containerization:** **Docker** & **Docker Compose** for standardized environment orchestration across development and production.
- **Validation Layer:** Custom-built **Validator** middleware for granular schema validation of Request Bodies, Parameters, and Query strings.
- **Dependency Injection:** Clean architecture implementation using internal **DI Containers** in each service to manage component lifecycles and testability.

## 🏗 Core Architecture (Shared Packages)

The `pkg/` directory serves as the backbone of the monorepo, providing unified standards across all services:

- **Server & Router:** A Fiber server abstraction featuring built-in health checks, centralized error handling, and graceful shutdown mechanisms.
- **Events:** An asynchronous communication system supporting Event-Driven Architecture (EDA).
- **Redis & Shared Storage:** Integrated Redis client handling the cache layer, distributed locks, and configuration constants.
- **Shared Middleware:** Pre-configured components for Rate Limiting, structured logging, cryptographic operations, and UUID validation.
- **Errors:** A standardized application error structure that automatically attaches request metadata for easier debugging.

## 🚀 Services Overview

The system is composed of specialized microservices, each responsible for a specific business domain:

| Service                  | Responsibility & Key Features                                                                                           |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------- |
| **API Gateway**          | Entry point (Reverse Proxy), CORS management, Helmet security, Redis session validation, and traffic aggregation.       |
| **Auth Service**         | Identity management, JWT issuance, password resets, Refresh Token rotation, and user device registration.               |
| **Audit Service**        | Critical event logging. Uses **SQLC** for optimized database access and internal workers for background log processing. |
| **Notification Service** | Asynchronous notification delivery utilizing a Worker/Service pattern for background task processing.                   |
| **Citizen Docs**         | Management of citizen documentation. Implements a clean repository pattern with GORM/PostgreSQL support.                |
| **Version Service**      | Orchestrates platform versioning and maintains compatibility across system components.                                  |

## 🛡 Security & Observability

- **Structured Logging:** Each service generates a single, high-performance app.log stream that combines network metrics (latency, status) with application events, featuring automatic reflection-based masking for sensitive data like passwords and tokens.
- **Request Tracking:** Full propagation of `X-Request-Id` across service boundaries for end-to-end request tracing.
- **Validation Layer:** Rigorous input validation at the middleware level, ensuring data integrity before reaching the business logic.
- **Security Headers:** Automated protection via Helmet middleware and dedicated CORS policies enforced at the Gateway level.
