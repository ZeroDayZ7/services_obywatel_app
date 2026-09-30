# Obywatel Platform - Microservices Monorepo

This repository contains the core backend platform for citizen-facing digital services. It is organized as a Go monorepo with shared infrastructure packages and multiple service modules operating under a common runtime and security model.

## Platform Scope

The platform provides a layered architecture for identity, communication, document handling, auditability, and administrative workflows. Shared code is centralized under the `pkg/` directory to keep service behavior consistent across the monorepo.

## Core Services

- **Gateway** - public entry layer for routing, request validation, session handling, and platform-level access control.
- **Auth / Auth Service** - authentication, authorization, account lifecycle operations, and token-based security flows.
- **Identity Service** - user identity management, registration operations, and identity-related worker processing.
- **Audit Service** - structured event logging, operational tracing, and asynchronous processing of critical records.
- **Notification Service** - notification persistence and background processing for user-facing alerts.
- **Messaging Service** - internal messaging and event-driven communication patterns across services.
- **Citizen Docs** - document management and citizen-related record handling.
- **Delegation Service** - delegation and authorisation workflow management.
- **Voting Service** - election and voting domain workflows.
- **Tally Service** - result aggregation and tabulation processing.
- **Version Service** - service and compatibility versioning across the platform.
- **Document Renderer** - rendering and generation of documents used by the platform.
- **Admin BFF / Officer BFF** - administrative and officer-facing backend interfaces.

## Shared Runtime Components

The platform uses a common Go-based runtime and infrastructure library across services:

- **Fiber-based HTTP layer** for service routing and request handling.
- **PostgreSQL** for transactional persistence in domain services.
- **Redis** for session storage, cache operations, and distributed coordination.
- **Docker and Docker Compose** for environment isolation and service orchestration.
- **Zap-based structured logging** for operational visibility and traceability.
- **DI containers** for service composition and lifecycle management.

## Security Model

The platform uses a service-level security model based on internal HMAC validation and KMS-backed key management.

- **KMS integration** is used for key provisioning and key lifecycle control.
- **HMAC validation** is applied for internal service-to-service integrity checks.
- **Rotating credentials** are managed through the KMS flow and are part of the current credential rotation process.
- **Redis and worker-based processing** remain controlled by environment configuration, with local placeholder behavior used where asynchronous processing is intentionally disabled.

## Observability and Integrity

- request identifiers are propagated across service boundaries
- structured logs capture operational state and application events
- validation middleware enforces input rules before core handlers execute
- service boundaries are designed for clear ownership, traceability, and safer integration

## Repository Structure

The monorepo is organized around a shared platform layer and service-specific modules under the `platform/` tree, with each service maintaining its own configuration, handlers, repositories, and runtime dependencies.
