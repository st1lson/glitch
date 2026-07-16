<div align="center">
  <img src="assets/logo.svg" alt="Glitch Logo" width="300" />
  <p><strong>API Chaos Engineering. Stop pretending your backend is perfect.</strong></p>
</div>

---

Most local development and E2E testing environments give you a perfect, zero-latency utopia. But in the real world, backends are flaky, networks drop packets, and servers randomly throw `502 Bad Gateway` errors. 

**Glitch** is a local development interceptor built specifically to simulate the chaos of production. Instead of manually hardcoding fake error responses into your apps to test loading states and error boundaries, you let Glitch inject chaos deterministically. 

It wraps your API—either by reverse-proxying your staging environment or mocking it locally—and intentionally breaks things so you can build resilient applications and robust E2E test suites (Cypress, Playwright, etc.).

---

## 🌪️ Chaos Engineering (The Fun Part)

Glitch acts as a configurable middleware that sits between your client applications (frontend, mobile apps, or other backend microservices) and the API.

### 1. Simulating Latency
Test your loading spinners, skeleton UIs, and network timeout logic deterministically.

```bash
# Add exactly 2 seconds to every request
glitch --proxy https://api.staging.com --latency 2s

# Simulate a variable normal distribution between 500ms and 3s
glitch --proxy https://api.staging.com --latency normal:500ms,3s
```

### 2. Injecting Failures
Ensure your error boundaries, retry mechanisms, and circuit breakers actually work.

```bash
# Force 20% of all requests to fail randomly
glitch --proxy https://api.staging.com --fail-rate 20

# Force 10% to be 429 Too Many Requests, and 5% to be 503 Service Unavailable
glitch --proxy https://api.staging.com --status 429:10,503:5
```

### 3. Bandwidth Throttling
Simulate a poor 3G connection by capping download speeds. Instead of just delaying the response, Glitch streams the payload to the consumer in tiny chunks.

```bash
# Cap download speed to exactly 50 kilobytes per second
glitch --proxy https://api.staging.com --bandwidth 50kbps

# Dial-up speeds
glitch --proxy https://api.staging.com --bandwidth 5kb/s
```

### 4. Payload Corruption (Schema Resilience)
Verify your frontend's resilience against schema drifts, unexpected null values, or missing fields by corrupting response payloads. Configured via profile or global config, Glitch intercepts JSON payloads and mutates their contents dynamically.

```yaml
# glitch.yaml
corruption:
  rate: 15             # Corrupt 15% of JSON response payloads
  strategies:          # Optional: choose specific mutators
    - drop_field       # Drop random field from objects
    - swap_type        # Change data types of values
    - inject_null      # Replace value with null
    - break_syntax     # Mess up raw JSON format
  multi: true          # Apply multiple mutators at once
```

### 5. Shareable Chaos Profiles
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
stall:
  rate: 5
  mode: drop
corruption:
  rate: 10
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
*Glitch completely bypasses CORS restrictions and ignores self-signed TLS errors, letting your local `localhost` apps or E2E tests seamlessly consume remote APIs.*

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

## 🛠️ Global Configuration

Tired of typing long CLI flags? You can define your entire testing environment natively via a `glitch.yaml` file. Glitch automatically discovers `glitch.yaml` or `.glitch.yaml` in your current working directory.

```yaml
# glitch.yaml
port: 8080
proxy: http://api.staging.internal
verbose: true
latency:
  distribution: "normal"
  min: 200ms
  max: 2s
failure:
  rate: 15
  statuses:
    - code: 502
      rate: 10
bandwidth: 50kbps
stall:
  rate: 5
  mode: drop
  drop_at: 50
corruption:
  rate: 10
  strategies:
    - drop_field
    - inject_null
  multi: false
```

Now you can simply run:
```bash
glitch
```

*Note: CLI flags always take precedence over the global configuration file, allowing you to easily override settings on the fly.*

---

## Installation

### Via Go
Ensure you have Go 1.20+ installed, then run:

```bash
go install github.com/st1lson/glitch/cmd/glitch@latest
```

### Via Docker
Glitch is automatically published to the GitHub Container Registry as a highly optimized, tiny Alpine image.

```bash
docker run -p 3000:3000 ghcr.io/st1lson/glitch:latest --proxy https://api.staging.com --fail-rate 10
```

Perfect for dropping into a `docker-compose.yml` stack to run your Playwright / Cypress tests against a chaotic local backend!

---

## CLI Reference

```text
Usage:
  glitch [file] [flags]

Flags:
      --bandwidth string   throttle response bandwidth (e.g. "50kbps", "1mbps")
      --config string      path to global config file (default: auto-discovers glitch.yaml)
      --fail-rate string   Overall failure rate percentage (e.g., 20)
  -h, --help               help for glitch
      --host string        Host to bind to (default "localhost")
      --latency string     Inject latency (e.g., 2s, normal:500ms,2s, uniform:1s,3s)
      --no-tui             disable the interactive dashboard and use standard stdout logging
  -p, --port int           Port to listen on (default 3000)
      --profile string     Name of a chaos profile to apply
      --proxy string       Proxy requests to this target URL instead of using a local file
      --read-only          Do not persist changes to the JSON database
      --status strings     Comma-separated specific status failures (e.g., 500:10,429:5)
  -v, --verbose            Enable verbose logging (prints request/response bodies)
```