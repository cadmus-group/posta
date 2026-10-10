# Security policy

Posta is infrastructure software; deployments must be secured by their operators. Its example Compose and gateway configurations are **not** a production security baseline.

## Reporting a vulnerability

**Do not create a public issue or pull request containing an exploit, credentials, or sensitive details.**

Use GitHub's private vulnerability reporting / repository security advisory flow when it is enabled: [Report a vulnerability](https://github.com/cadmus-group/posta/security/advisories/new). If that option is unavailable, contact a repository maintainer privately using an available contact channel on their GitHub profile before disclosing details. Do not assume private reporting is enabled.

Include the affected version or commit, relevant configuration, impact, steps to reproduce, and a safe proof of concept where possible. Maintainers will triage in good faith; no fixed response or remediation timeline is promised.

## Supported versions

Security fixes are considered for the latest maintained release and current `main` branch, subject to maintainer availability. Older releases are not guaranteed to receive backports.

## Operational boundaries

- Do not expose Redis, Asynq monitoring, or unauthenticated enqueue endpoints publicly.
- Configure `ENQUEUE_AUTH_TOKEN`, or preferably enforce stronger authenticated ingress and network controls for production.
- The queue API accepts a shared static token; it does **not** verify JWT claims or perform tenant isolation.
- Keep task payloads free of secrets when using the built-in `generic` example handler, which logs payloads.
- Review the external Traefik rate-limit plugin, its Redis credentials, and the actual routes protected by it.
- Provision TLS, secrets, access controls, rate limiting, and observability based on your environment.
- Treat any credential-like values ever committed to version control as potentially exposed; replace and rotate them before publishing a repository or deploying a public-facing instance.

See the [README](README.md) for the boundary between local examples and production responsibilities.

