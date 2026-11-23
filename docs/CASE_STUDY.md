# Case study: Mirohodec Warden

## Context

A Telegram community needed the same bot to handle moderation and social interaction. The author reports running it on a VPS and using it in a real, commercial/community setting. This showcase is based on source and configuration found in an archived codebase; no user-base, SLA or revenue number has been inferred.

## Engineering decisions

**Problem 1: community policy is not a single ban command.** Warnings expire, multiple chats have different configuration, and automatic action must respect protected users and prior moderation state. The application tracks moderation records and evaluates escalation. A pure decision module makes this policy individually testable.

**Problem 2: message floods are diverse.** A strict universal messages-per-second cap produces false positives. The original system records independent streams for text, media, stickers, links, mentions, forwards and repeated signatures. Cooldowns and strike severity mediate the response.

**Problem 3: a game has long-lived state.** Mafia is not a stateless bot command. Registration, rooms, roles, timed phases, votes, cancellations, ELO and private game information need consistent persistence. Service functions coordinate transitions and the store captures state between events.

**Problem 4: operators need more than Telegram commands.** A Go HTTP panel exposes multi-group configuration, warnings, users, content and game endpoints. The server is separate from the Telegram command adapter, with its own session validation.

**Problem 5: the bot is also a character.** Laya has configurable prompts, response context, cooldowns, quality filters and content tools. A remote AI API is optional, so baseline moderation does not depend on an external model being available.

## Technical vocabulary demonstrated in this repository

Go; interfaces; dependency selection; Telegram Bot API; WebSocket; REST-style HTTP endpoints; PostgreSQL; SQL migrations; JSON persistence; structured asynchronous logging; Prometheus metrics; external AI HTTP integration; embedded resources; automated tests; Docker; CI.

## Scope and provenance

This is an adaptation of the author's existing backend. The source proves implemented components and provides runnable interfaces and tests. It does **not** prove traffic, unique user count, profitability or CI results prior to publication. Those numbers should be added only when backed by logs or public evidence.

## A concise portfolio description

> Designed and maintained a Go Telegram backend combining multi-chat moderation, configurable anti-spam, an AI-powered social character and a stateful Mafia game. Implemented JSON/PostgreSQL persistence, a web control panel, HTTP/WebSocket game endpoints, structured logs, metrics, configuration reload and automated tests. Deployed the original application to a VPS and later prepared a sanitized, documented public showcase.
