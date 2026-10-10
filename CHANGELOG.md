# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]
### Removed
- Demo/example tasks, bundled demo scheduler job, and hardcoded smoke test payloads. The stack no longer ships with the former application-specific demonstration tasks or scheduled jobs.
### Changed
- A built-in `generic` sample module is available for local testing; production tasks still require user-defined handlers.

## [0.1.0] - 2026-01-10
### Added
- Initial edge + jobs stack: queue-api, worker, scheduler, Traefik gateway, Redis-backed rate limiting, and asynqmon UI.
- Token-gated `/v1/enqueue` with task type allowlist and payload validation.
- Example tasks (ping/email), demo scheduler job, and smoke test script.
- Docker Compose dev stack and distroless images for all services.
- Version metadata injection, Makefile build targets, and `.env.example` defaults.

