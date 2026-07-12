# Glitch: Frontend Chaos Engineering

Most local development environments are perfect. Your local API resolves in 2ms, zero packets are dropped, and you never hit a `502 Bad Gateway` error. But production is chaotic.

**Glitch** is a local development interceptor built to simulate the chaos of production. It sits between your frontend application (React, Vue, Angular, etc.) and your backend API, injecting artificial latency, throttling bandwidth, and randomly failing requests so you can test your loading states, error boundaries, and retry logic.

## Why Glitch?

- **Test Error Boundaries:** Automatically throw 5xx or 4xx errors at random to ensure your frontend handles failures gracefully.
- **Test Loading States:** Simulate slow backends using fixed or variable (normal/uniform distribution) latency.
- **Test Progressive Rendering:** Throttle bandwidth (e.g. `50kbps`) to test how large JSON payloads or images load on poor 3G connections.
- **Team Alignment:** Share "Chaos Profiles" via YAML so your entire QA and engineering team can test against the exact same degraded network conditions.

## Core Engines

Depending on your backend setup, Glitch runs in three different modes:

1. **Reverse Proxy:** Don't have a local backend running? Proxy your staging server. Glitch handles the CORS bypassing.
2. **OpenAPI Mock:** Give Glitch a `swagger.yaml` or `openapi.yaml` file, and it will auto-generate realistic JSON responses.
3. **JSON Database:** Just have a `db.json` file? Glitch will generate a fully functioning REST API (CRUD + Pagination + Sorting) instantly.

## Next Steps

- [Chaos Engineering Setup](./chaos.md)
- [Interceptor Engines](./engines.md)
- [Global Configuration](./configuration.md)
- [Docker & CI Integration](./docker-ci.md)
