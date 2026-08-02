# Messaging Service

Stateless microservice handling end-to-end encrypted (E2EE) messaging, delta synchronization, and cryptographic key management. Built with Go, Fiber, and GORM (PostgreSQL).

---

## Features

* **End-to-End Encryption (E2EE)**
* Stores and serves device identity keys (`UserDeviceIdentity`).
* Distributes one-time public keys (`UserPreKey`) for X3DH / Signal Protocol session initialization.
* Processes encrypted binary payloads (`EncryptedPayload` / AES-GCM) without backend access to message content.


* **Delta Sync Engine**
* Uses monotonic versions (`Version`) and conversation sequence numbers (`Sequence`).
* Synchronizes message and contact states for mobile clients (Drift/SQLite).


* **Relationship & Chat Management**
* Manages contact lists with locally encrypted aliases.
* Supports direct and group conversations.


* **Security & Operations**
* Validates internal requests via HMAC middleware (`InternalAuthMiddleware`).
* Built using standalone exported functions.
* Includes database migration support and development seed data.
* Containerized using `distroless/static-debian12` running as non-root.