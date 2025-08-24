# About This Project

This document outlines the core mission, intent, and operational principles of the Hugging Face Scraper Daemon.

## Project Mission

The primary mission of this service is to create and maintain an autonomous, "living mirror" of Hugging Face model metadata. It operates as a persistent, background daemon designed to build a complete local database and keep it continuously up-to-date with minimal human intervention.

The goal is to provide a fast, reliable, and local data source that can be used to power other applications, dashboards, or analytical tools, without needing to query the public Hugging Face API for every request.

## The "Smart" Daemon: Operational Principles

This service is designed to be more than just a script; it is an intelligent agent that understands its own state and operates in two distinct modes.

### 1. Backfill Mode: The Sprint

When the service starts for the first time, it enters **Backfill Mode**.

- **Objective:** To build a complete, historical database of every model on Hugging Face.
- **Process:** It systematically fetches all models by their creation date (`createdAt`), starting from the oldest, and saves each one to the local database.
- **Completion:** This intensive, one-time process is complete when the last page of historical models has been ingested.

### 2. Watch Mode: The Marathon

Once the backfill is complete, the service automatically transitions into its permanent steady state: **Watch Mode**.

- **Objective:** To keep the database fresh with maximum efficiency and minimal API usage.
- **Process:** On a regular, configurable interval (e.g., every 5 minutes), the daemon performs a highly efficient check for any changes:
  1.  It first queries its own database to find the timestamp of the most recently modified model it has stored.
  2.  It then asks the Hugging Face API for the latest models, sorted by `lastModified`.
  3.  It compares the incoming models against its latest known timestamp. For any model that is newer, it performs an "upsert" operation—updating the record if it exists, or creating it if it's new.
  4.  This process stops as soon as it encounters a model that is not newer than its latest known record, avoiding unnecessary API calls.

> **How New Models Are Detected:**
> This single process smartly handles both updates to existing models and the creation of brand new models. When a new model is created, its `lastModified` timestamp is set to its `createdAt` timestamp. This means that a sort by `lastModified` naturally includes brand new models at the top of the list, allowing the daemon to detect both updates and creations in a single, efficient query.

### 3. Stateful Resilience

The service is **stateful**. It records its current operational mode (e.g., `NEEDS_BACKFILL` or `WATCHING`) in the database. This ensures that if the service is restarted, it can intelligently resume its work without starting the intensive backfill process all over again.

### 4. Read-Only Data Access

The API and Web UI provided by this service are intended as a **read-only window** into the curated database. They do not trigger any scraping operations. Their purpose is to serve the data that the background daemon has worked to prepare.

For more technical details on the layered design, see the `STRUCTURE.md` file.
