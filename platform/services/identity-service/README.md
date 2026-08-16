# Identity Service

## Overview

The Identity Service is a core microservice responsible for managing user identity records, personal profile data, and secure identification across the platform.

## Purpose

It acts as the single source of truth for user personal data, ensuring strict confidentiality, compliance, and cryptographic protection of sensitive identifiers before persistent storage.

## Key Functions

- User Identification: Manages core user profile attributes and personal identity records.
- Envelope Encryption: Secures sensitive data fields at rest using envelope encryption backed by the Key Management Service (KMS) with DEK/KEK rotation patterns.
- Blind Indexing: Enables deterministic lookup on encrypted attributes (e.g., PESEL) using salted cryptographic hashing without exposing plain text values.
- Profile Operations: Exposes internal APIs for downstream services to query and update verified identity states.
