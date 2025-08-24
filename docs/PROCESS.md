# Application Process Flow

This document provides a detailed, step-by-step explanation of the `hf-scraper` daemon's operational flow. It covers the logic from initial startup through its two primary modes: Backfill and Watch.

For a higher-level overview of the project's mission, see `ABOUT.md`.
For the architectural layout, see `STRUCTURE.md`.

---

## 1. Initial Startup and State Check

When the daemon is launched (`go run cmd/daemon/main.go`), it follows a precise startup sequence to determine which operational mode to enter.

```plaintext
[ Start Daemon ]
       |
       v
[ Load Configuration ]
       |
       v
[ Connect to Database ]
       |
       v
[ Read Service Status from DB ]
       |
       v
< Is status 'NEEDS_BACKFILL'? >
       |             |
     (YES)           (NO)
       |             |
       v             v
[ Enter Backfill Mode ] [ Enter Watch Mode ]
```

1.  **Load Config:** All necessary parameters (DB URI, rate limits, etc.) are loaded.
2.  **Connect to DB:** A connection to MongoDB is established.
3.  **Check State:** The daemon queries its status collection (e.g., `_status`) to read its last known state, including the `backfillCursor` if it exists.
4.  **Decision:**
    - If the state is `NEEDS_BACKFILL` (or does not exist), it proceeds to **Backfill Mode**.
    - If the state is `WATCHING`, it skips the backfill entirely and proceeds directly to **Watch Mode**.

---

## 2. Process Flow: Backfill Mode

The goal of this mode is to perform a one-time, comprehensive scrape of all historical models. This process is designed to be **highly resilient** and **performant**.

**Start Signal:** The service enters this mode if its state is `NEEDS_BACKFILL`.

**Process Loop:**

1.  Initialize the scraper with the starting API endpoint (`/api/models?sort=createdAt`) or the saved `backfillCursor` URL.
2.  **Fetch** the current page of models from the API.
3.  **Store Models:** Perform a single, high-performance **Bulk Upsert** operation to store all models from the fetched page. This is idempotent, making the process safely resumable.
4.  **Update Cursor:** **Only after the models are successfully stored**, the daemon updates the `backfillCursor` in its status document to the `rel="next"` URL from the API response's `Link` header.
    > This transactional order (Store Models -> Update Cursor) is critical. If the service crashes after storing but before updating the cursor, it will simply re-process the same page on restart, ensuring no data is lost.
5.  **Continue Signal:** If a `rel="next"` URL exists, the loop continues using this new URL.
6.  **Stop Signal:** If the `Link` header **does not** contain a `rel="next"` URL, the daemon has reached the last page. The loop terminates.

**On Completion:**

1.  The daemon updates its state in the status collection to `WATCHING`.
2.  It publishes a `status:mode_change` event to the internal event broker.
3.  It immediately transitions into **Watch Mode**.

---

## 3. Process Flow: Watch Mode

The goal of this mode is to efficiently keep the database up-to-date with the latest changes. It runs in a continuous loop on a configured timer (e.g., every 5 minutes).

**Start Signal:** The service enters this mode if its state is `WATCHING` or after the Backfill Mode completes.

**A Single Watch Cycle:**

1.  **Establish Anchor:** The cycle begins by querying its **own database** to find the timestamp of the model with the most recent `lastModified` date. This timestamp becomes the `latestKnownUpdate` benchmark for this cycle.
2.  **Fetch Latest:** The scraper makes a single API call to fetch the **first page only** of `/api/models?sort=lastModified`.
3.  **Collect Updates:** The daemon loops through the list of models returned by the API and compares each model's timestamp to the benchmark.
4.  **Continue Signal:** If `model.lastModified > latestKnownUpdate`, it means the model is new or has been updated. The model is added to a temporary list of updates.
5.  **Stop Signal:** The moment the loop encounters a model where `model.lastModified <= latestKnownUpdate`, it immediately **stops iterating**. Because the API results are sorted, this signal guarantees that all subsequent models are not new.
6.  **Store Updates:** If the list of updates is not empty, the daemon performs a single **Bulk Upsert** operation to store all the new/updated models efficiently.
7.  **Sleep:** The watcher now waits for the next timer tick to start the cycle over again.
