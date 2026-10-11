# WebSocket Service

Centralized real-time notification and connection management service for the platform.

## Overview

The `websocket-service` acts as a real-time communication hub. It maintains persistent WebSocket connections with client applications and instantly pushes notifications, messages, and state updates as they occur, avoiding unnecessary polling.

## Architecture & Flow

* **Client Connections:** Clients establish a secure WebSocket connection (upgraded from HTTP). Connections are mapped to specific user IDs upon authentication.
* **Internal Event Distribution:** The service subscribes to **Redis Pub/Sub** channels. When other backend services (e.g., messaging or notification services) generate an event for a user, they publish it to Redis.
* **Real-time Push:** The WebSocket service receives the event from Redis, looks up the user's active connection in the local Connection Manager, and delivers the payload instantly.

## Tech Stack

* **Language:** Go
* **Framework:** Fiber v3
* **Broker:** Redis Pub/Sub
* **Transport:** WebSockets (`fasthttp/websocket`)