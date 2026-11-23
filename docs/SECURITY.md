# Security and publishing notes

## Before publishing

1. **Rotate the Telegram Bot API token and OpenRouter API key from the archived `.env`.** They were stored alongside production source; do not assume that excluding `.env` from this new ZIP makes the original keys safe.
2. Never import the original `.git` directory/history, `.deploy` backups, personal media, raw user state, production logs or `config.yaml` into the new public repository.
3. Verify that you have permission to release business-specific code and assets if the project was operated for a client or employer.
4. Copy `.env.example` into local `.env` only on your machine. `.gitignore` blocks `.env`; check staged files before every push.
5. Run `python3 scripts/check_public.py` locally and in CI. The checks are intentionally conservative, but are not a substitute for a dedicated secret scanner.

## Deployment

- Turn off `DEV_AUTH` outside isolated local development. It is a developer-only bypass for Mini App authentication.
- Keep the panel behind TLS; use a long random `WEB_ADMIN_TOKEN`, secure cookies and correct proxy settings.
- Restrict network access to metrics and PostgreSQL; do not expose database/metrics ports publicly.
- Keep per-chat admin IDs and permissions minimal. The bot should never accept raw user text as an authorization grant.
- Use a dedicated Telegram token, avoid logging authentication data or raw model credentials, and use secret injection on the host.
- Review request size/timeouts, session behavior and rate limits before opening the panel to untrusted networks.

## Disclaimer

The conversion kept existing application behavior wherever practical. It has **not** been independently penetration-tested, load-tested or verified against real Telegram traffic in this environment. An employer-facing showcase should state this plainly.
