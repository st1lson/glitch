# Dockerization & CI Integration 🐳

Glitch isn't just for local frontend developers. It is built to be deployed seamlessly into CI/CD pipelines to ensure your frontend tests (e.g. Cypress, Playwright) actually test how resilient your UI is under degraded conditions.

## Running via Docker

Glitch is automatically published to the GitHub Container Registry (`ghcr.io`) as a highly optimized, statically compiled Alpine Linux image.

```bash
docker run -p 3000:3000 ghcr.io/st1lson/glitch:latest --proxy https://api.staging.com --fail-rate 10
```

> **Note:** The Docker image includes `ca-certificates` by default, meaning you can flawlessly proxy to HTTPS staging servers without worrying about x509 certificate errors inside the container.

## CI/CD Service Container (Playwright / Cypress)

If your QA team writes End-to-End tests, they typically test the "happy path" against a perfect database.

Instead, you can drop Glitch into your `docker-compose.yml` file, route the frontend to talk to Glitch, and have Glitch proxy the real backend. You can then run your Cypress tests against Glitch to verify error boundaries.

```yaml
version: "3.8"
services:
  frontend:
    image: my-react-app
    ports:
      - "80:80"
    environment:
      # Tell the frontend to hit Glitch instead of the real API
      - REACT_APP_API_URL=http://glitch:3000

  # Glitch interceptor running in chaos mode
  glitch:
    image: ghcr.io/st1lson/glitch:latest
    ports:
      - "3000:3000"
    command: ["--proxy", "https://api.staging.internal", "--latency", "normal:500ms,2s", "--no-tui"]

  # Your actual backend
  backend:
    image: my-backend-api
```

### Important CI Flag: `--no-tui`
When running in Docker or CI pipelines, be sure to pass the `--no-tui` flag. Glitch detects when it is running without a TTY, but explicitly passing `--no-tui` forces Glitch to disable the interactive dashboard and stream standard structured logs to `stdout`, which makes debugging CI failures much easier.
