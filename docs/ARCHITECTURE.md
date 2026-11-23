# Architecture and engineering notes

This repository is a production-history backend adapted for public review. Some boundaries are legacy; the documentation distinguishes existing boundaries from the direction of the refactor.

## Executable boundaries

1. `cmd/warden/main.go` is the composition root: config, logger, repository selection, Telegram transport, workers, metrics and admin HTTP server lifecycle.
2. `RuntimeStore`, `PhotoRepository`, `QuoteRepository`, and `FAQRepository` are contracts implemented by JSON and PostgreSQL storage adapters. The application does not require PostgreSQL to run locally.
3. Mafia game transitions live in `MafiaService` and repository state, not the HTML templates. The WebSocket/HTTP layer in `internal/web` uses a service interface.
4. `internal/domain/moderation` is a pure, deterministic decision module. The handler consults it before executing Telegram API calls.
5. Laya's external text generation uses an HTTP service with configurable timeout, fallback and prompt/context processing. The AI model is not hosted by this repository.
6. Translation maps are embedded JSON; migrations are embedded, numbered SQL statements. Both are versioned alongside source.

## Important invariant examples

- An admin or protected account should not be moderated by automated actions if protection is enabled.
- A warning action must be derived from the active warning count; topic restriction precedes mute at the stricter threshold.
- A Mafia night action must be accepted only for the appropriate role/phase; secret roles must not be disclosed to other players.
- Cancelled games and finished games must be accounted for separately.
- Telegram Mini App authentication depends on server-side verification of signed init data; client input alone is not trusted.
- Each chat has independent policy and state. Actions must be scoped by chat ID and user ID.

## Public refactor performed

- Original monolithic Go files were split by top-level declarations without changing the declarations' meaning, to comply with the 200-line convention.
- `LoadConfig` was decomposed into defaults/validation helpers, still preserving the original order of application.
- The long Mafia callback dispatcher and spam action branch were separated into explicit functions.
- The long in-app Mafia diagnostic was separated into setup, action and cancellation checks.
- Localized strings and SQL DDL were extracted into embedded structured data.
- Warning escalation policy was moved behind an importable Go package and covered by table tests.
- Credentials, `.git`, local state, real user IDs, photos and other operational artifacts were removed from the public archive.

## Known architectural debt

- Most application logic still shares `package main`. Smaller files do not make it a textbook layered architecture.
- Database migrations are idempotent but lack a dedicated `schema_migrations` version table. For a larger team, use transactional numbered migrations with version tracking.
- The original app uses business-specific naming for the forum-topic restriction mechanic; public documentation uses neutral terminology.
- Original tests include high-level behavior checks, but live Telegram interactions and production load must be tested against a staging deployment.
- AI output is probabilistic; safety and performance claims require ongoing monitoring rather than simply calling an LLM API.

## Suggested next slice

Move Mafia transitions and repository interfaces into `internal/application/mafia` and `internal/domain/mafia`, define domain events for game state changes, and make persistence transactional where several user/game records are updated together. Migrate by vertical slice with tests rather than a giant rewrite that would make behavior hard to verify.
