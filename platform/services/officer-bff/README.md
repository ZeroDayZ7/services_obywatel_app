# Officer BFF Service

## Overview

The Officer BFF (Backend-for-Frontend) serves as an API aggregation layer specifically designed for the Officer Portal application. It acts as an adapter between the client-facing interface and internal platform microservices.

## Purpose

Instead of requiring the frontend to orchestrate calls across multiple domain services, the BFF consolidates downstream responses into UI-tailored models. This design isolates client logic, minimizes round-trip latency, and simplifies portal data management.

## Key Functions

- Data Aggregation: Combines information from multiple core microservices (such as Identity, Auth, and Citizen Docs) into unified responses.
- User Onboarding: Processes registration workflows for new officer accounts.
- Profile Management: Handles updates and modifications to user administrative records.
- Payload Optimization: Shapes and filters backend responses to match frontend requirements.

## Frontend Integration

This service exposes a dedicated REST API designed to power the Angular 22 Officer Portal web application.
