<div align="center">
  <img src="assets/logo.svg" alt="Glitch Logo" width="300" />
  <p><strong>Frontend Chaos Engineering. Stop pretending your backend is perfect.</strong></p>
</div>

---

Most local development environments give you a perfect, zero-latency utopia. But in the real world, backends are flaky, networks drop packets, and servers randomly throw `502 Bad Gateway` errors. 

**Glitch** is a local development interceptor built specifically to simulate the chaos of production. Instead of manually hardcoding `setTimeout` or fake error responses into your React/Vue components to test loading states and error boundaries, you let Glitch inject chaos deterministically. 

It wraps your API—either by reverse-proxying your staging environment or mocking it locally—and intentionally breaks things so you can build resilient frontends.

---

## 🌪️ Chaos Engineering (The Fun Part)

Glitch acts as a configurable middleware that sits between your frontend and the API.

### 1. Simulating Latency
Test your loading spinners, skeleton UI, and timeout logic without throttling your entire browser via DevTools.

```bash
# Add exactly 2 seconds to every request
glitch --proxy https://api.staging.com --latency 2s

# Simulate a variable normal distribution between 500ms and 3s
glitch --proxy https://api.staging.com --latency normal:500ms,3s
```

### 2. Injecting Failures
Ensure your error boundaries, toast notifications, and retry mechanisms actually work.

```bash
# Force 20% of all requests to fail randomly
glitch --proxy https://api.staging.com --fail-rate 20

# Force 10% to be 429 Too Many Requests, and 5% to be 503 Service Unavailable
glitch --proxy https://api.staging.com --status 429:10,503:5
```

### 3. Shareable Chaos Profiles
Save your worst-case scenarios as YAML files and commit them to your repository (`.glitch/profiles/flaky.yaml`) so your whole team can test against the same chaotic conditions.

```yaml
latency:
  distribution: "normal"
  min: "1s"
  max: "4s"
failure:
  rate: 30
  statuses:
    - code: 502
      rate: 15
```

Run it instantly:
```bash
glitch --proxy https://api.example.com --profile flaky
```

---

## 🔌 Core Interceptor Modes

Glitch needs an API to wrap its chaos around. It provides three robust engines depending on what you have available:

### Reverse Proxy (Recommended)
Don't have a local backend? Point Glitch at your live staging environment.
```bash
glitch --proxy https://api.mycompany.staging.com
```
*Glitch completely bypasses CORS restrictions and ignores self-signed TLS errors, letting your local `localhost` frontend seamlessly consume remote APIs.*

### OpenAPI Mock Server
Have an OpenAPI v3 spec? Let Glitch mock it locally.
```bash
glitch api.yaml
```
*Glitch dynamically generates fake JSON responses based on your schemas and types.*

### Generic JSON Database
Don't even have a spec yet? Just pass a JSON file.
```json
{ "users": [{ "id": 1, "name": "Alice" }] }
```
```bash
glitch db.json
```
*You instantly get a full REST CRUD API (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`) with built-in sorting, filtering, and pagination.*

---

## Installation

Ensure you have Go 1.20+ installed, then run:

```bash
go install github.com/st1lson/glitch/cmd/glitch@latest
```

---

## CLI Reference

```text
Usage:
  glitch [file] [flags]

Flags:
      --fail-rate string   Overall failure rate percentage (e.g., 20)
  -h, --help               help for glitch
      --host string        Host to bind to (default "localhost")
      --latency string     Inject latency (e.g., 2s, normal:500ms,2s, uniform:1s,3s)
  -p, --port int           Port to listen on (default 3000)
      --profile string     Name of a chaos profile to apply
      --proxy string       Proxy requests to this target URL instead of using a local file
      --read-only          Do not persist changes to the JSON database
      --status strings     Comma-separated specific status failures (e.g., 500:10,429:5)
  -v, --verbose            Enable verbose logging (prints request/response bodies)
```