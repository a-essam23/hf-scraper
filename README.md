# hf-scraper

An autonomous, stateful daemon that creates and maintains a local mirror of Hugging Face model metadata, exposed via a read-only REST API.

This service is designed to run continuously, first performing a full historical backfill of all models, and then switching to an efficient "watch mode" to keep the database up-to-date with the latest changes.

## Core Features

- **Autonomous Operation:** A "set it and forget it" daemon that manages its own data acquisition lifecycle.
- **Smart Backfill & Watch Modes:** Intelligently performs a one-time historical scrape, then switches to a permanent, efficient update-watching mode.
- **Stateful & Resilient:** Remembers its operational state across restarts to avoid unnecessary re-scraping.
- **Rate-Limited & Retry-Enabled:** Respectfully interacts with the Hugging Face API with built-in rate limiting and resilience to network errors.
- **Read-Only REST API:** Provides a fast and simple API to query the locally mirrored data.
- **Clean, Layered Architecture:** Designed for maintainability and clarity.

## Getting Started

Follow these steps to get the `hf-scraper` daemon running on your local machine.

### Prerequisites

- Go (version 1.21 or newer)
- MongoDB (version 7.0 or newer)
- Git

### Installation & Running

1.  **Clone the repository:**

    ```sh
    git clone https://github.com/your-username/hf-scraper.git
    cd hf-scraper
    ```

2.  **Create your configuration:**
    Copy the example config file.

    ```sh
    cp configs/config.yaml.example configs/config.yaml
    ```

    Now, **open `configs/config.yaml`** and edit the `DATABASE.URI` to point to your MongoDB instance.

3.  **Install dependencies:**

    ```sh
    go mod tidy
    ```

4.  **Run the daemon:**
    ```sh
    go run cmd/daemon/main.go
    ```

The service will now start. If this is the first run, it will begin the "Backfill Mode" to scrape all historical models. This may take a considerable amount of time. Subsequent runs will start in "Watch Mode".

## Configuration

All application settings are managed in `configs/config.yaml`. These values can also be overridden by environment variables (e.g., `SERVER_PORT=9090`).

| Key                           | Type     | Description                                                                  |
| ----------------------------- | -------- | ---------------------------------------------------------------------------- |
| `SERVER_PORT`                 | `string` | The port for the read-only API server.                                       |
| `DATABASE.URI`                | `string` | **Required.** The full connection string for your MongoDB instance.          |
| `DATABASE.NAME`               | `string` | The name of the database to use.                                             |
| `DATABASE.COLLECTION`         | `string` | The name of the collection to store models in.                               |
| `SCRAPER.REQUESTS_PER_SECOND` | `int`    | The number of API requests to make per second.                               |
| `SCRAPER.BURST_LIMIT`         | `int`    | The number of requests allowed in a short burst.                             |
| `WATCHER.INTERVAL_MINUTES`    | `int`    | How often (in minutes) the service should check for updates in "Watch Mode". |

## API Usage

The service exposes a simple, read-only REST API to access the mirrored data.

### Get Model by ID

Retrieves a single model from the local database.

- **Method:** `GET`
- **Path:** `/models/{author}/{modelName}`

**Example:**

```sh
curl http://localhost:8080/models/google-bert/bert-base-uncased
```

**Example Response:**

```json
{
  "id": "google-bert/bert-base-uncased",
  "author": "google-bert",
  "sha": "...",
  "lastModified": "...",
  "hf_createdAt": "...",
  "private": false,
  "gated": false,
  "likes": 1618,
  "downloads": 2486716,
  "tags": ["fill-mask", "bert", "pytorch"],
  "pipeline_tag": "fill-mask"
}
```

## Project Internals

For a deeper understanding of the project's design and philosophy, please see the following documents:

- **[ABOUT.md](docs/ABOUT.md):** Explains the project's mission and the logic behind its "smart" operational modes.
- **[STRUCTURE.md](docs/STRUCTURE.md):** Provides a complete breakdown of the layered architecture and directory structure.
