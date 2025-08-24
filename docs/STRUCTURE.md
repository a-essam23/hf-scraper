# Application Structure

This document outlines the layered architecture for the Hugging Face Scraper, which operates as an **autonomous data pipeline daemon**.

The primary goal of this service is to create and maintain a "living mirror" of Hugging Face model metadata. It is a stateful, long-running process that intelligently manages its own data acquisition lifecycle. Secondary goals include exposing this data via a read-only API and broadcasting real-time status updates.

## Core Principles

1.  **Dependency Hierarchy:** Layers are built from the inside out. Core layers have no project-specific dependencies.
2.  **Stateful Operation:** The service is aware of and persists its operational state (e.g., Backfill vs. Watch Mode).
3.  **Pragmatic Decoupling:** We use an interface for the **database storage** to allow for swapping backends. We use a **concrete scraper implementation** directly in the service layer to prioritize development speed.
4.  **Event-Driven Communication:** Key state changes within the service are announced via an internal event broker, decoupling the source of the event from any components that need to react to it.

## Cross-Cutting Components

### The Event Broker

At the heart of our internal communication is a lightweight, in-memory event broker (`internal/events/`). This component follows a simple Pub/Sub pattern, allowing different parts of the application to communicate without being directly coupled. For example, the **Service Layer** can _publish_ an event when its mode changes, and the **Delivery Layer** can _subscribe_ to this event to push notifications to clients.

## The Foundational Layers (From Core to Edge)

---

### Layer 0: Foundation (Configuration)

The bedrock of the application. Initialized first, it provides parameters for all other layers.

- **Responsibility:** Load configuration.
- **Directory:** `internal/config/`

---

### Layer 1: Domain (The Language)

Defines the core data structures used throughout the application.

- **Responsibility:** Define `HuggingFaceModel`, `ServiceStatus`, and event-related structs.
- **Directory:** `internal/domain/`

---

### Layer 2: Integration (The Tools)

Contains concrete implementations for interacting with all external systems.

- **Responsibility:**
  - **Scraper:** Fetch data from the Hugging Face API.
  - **Model Storage:** Persist `HuggingFaceModel` data. Includes the `ModelStorage` interface.
  - **Status Storage:** Read and write the service's operational state.
- **Directories:** `internal/scraper/`, `internal/storage/`, `internal/service/storage.go`

---

### Layer 3: Service (The Engine)

The central orchestrator of the daemon's logic.

- **Responsibility:**
  - Implement the **Backfill** and **Watch** mode logic.
  - Coordinate the scraper and storage components.
  - **Publish events** to the broker when significant state changes occur (e.g., backfill completion).
  - Provide data-retrieval methods for the Delivery Layer.
- **Directory:** `internal/service/`

---

### Layer 4: Delivery (The Window)

The outermost layer that provides interfaces to the outside world.

- **Responsibility:**
  - **REST API:** Provide a read-only API for querying model data.
  - **Server-Sent Events (SSE):** **Subscribe to the event broker** and push real-time status updates to connected clients (e.g., for an HTMX dashboard).
  - This layer is a passive observer and broadcaster; it does not trigger any core logic.
- **Directory:** `internal/delivery/`

## Directory Structure

```plaintext
/hf-scraper
├── cmd/daemon/
│   └── main.go           // Entry point: Orchestrates workers, the API server, and the event broker.
│
├── configs/
│   └── config.yaml
│
├── internal/
│   ├── config/           // Layer 0: Configuration loading.
│   │
│   ├── delivery/         // Layer 4: Read-only API and Real-time event handlers.
│   │   ├── rest/
│   │   └── sse/          // New: For Server-Sent Events.
│   │
│   ├── domain/           // Layer 1: Core data structures.
│   │
│   ├── events/           // New: The cross-cutting event broker.
│   │   └── broker.go
│   │
│   ├── service/          // Layer 3 (Engine) & Layer 2 (DB Contract).
│   │   ├── service.go
│   │   └── storage.go
│   │
│   ├── storage/          // Layer 2: Concrete DB implementations.
│   │   ├── mongo_storage.go
│   │   └── status_storage.go
│   │
│   └── scraper/          // Layer 2: Concrete scraper implementation.
│
├── web/
│   ├── static/
│   └── template/
│
├── go.mod
├── go.sum
├── README.md
```
