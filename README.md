<div align="center">
  <img src="assets/logo.svg" alt="Glitch Logo" width="300" />
  <p><strong>API Chaos Engineering. Stop pretending your backend is perfect.</strong></p>
</div>

Most local development and end-to-end test environments are not realistic at all: there is no slow speed, no lost connections, no random server problems.

Production is different. Backends have bad days, networks are not reliable, and eventually something is going to return a `502 Bad Gateway`.

Glitch is a development tool for creating those kinds of problems on purpose. Instead of putting fake failures in your application whenever you want to test a loading state, retry process, or error boundary, Glitch adds them for you in a controlled and repeatable way.

It sits in front of your API—either by reverse-proxying a staging environment or making one locally—and makes requests act badly. That gives you a way to create stronger applications and more realistic end-to-end test suites with tools like Cypress and Playwright.

> **📖 Using Glitch in CI/CD?** Check the [E2E Testing Integration Guide](https://github.com/st1lson/glitch/blob/main/docs/e2e-testing.md) for Playwright and Cypress examples, including pausing requests and adding failures while a test is running.

### SDKs

If you are using Playwright, [`glitch-playwright`](https://github.com/st1lson/glitch-js/tree/main/packages/playwright) gives each test its unique chaos scenario and removes it afterward, so tests can still run safely in parallel:

```js
import { expect, test } from 'glitch-playwright';

test('shows an error toast when the API fails', async ({ page, glitch }) => {
  await glitch.fail(500);

  await page.goto('/dashboard');

  await expect(page.getByRole('alert')).toBeVisible();
});
```

[`glitch-core`](https://github.com/st1lson/glitch-/tree/main/packages/core) offers the same client without the Playwright-specific tools, which makes it useful for scripts and other test frameworks. Both packages are in [st1lson/glitch-js](https://github.com/st1lson/glitch-js).

---

## 🌪️ Chaos Engineering

Glitch works as middleware between your client—whether that's a frontend, mobile app, or another backend service—and the API it communicates with.

### 1. Creating Latency

Add delay so you can test loading indicators, skeleton UIs, timeout handling, and other slow-network behavior.

```bash
# Add 2 seconds to every request
glitch --proxy https://api.staging.com --latency 2s

# Simulate delay between 500ms and 3s using a distribution
glitch --proxy https://api.staging.com --latency normal:500ms,3s
```

### 2. Adding Failures

Add HTTP failures to ensure your error handling, retries, and circuit breakers work outside the happy path.

```bash
# Make 20% of all requests fail randomly
glitch --proxy https://api.staging.com --fail-rate 20

# Return 429 for 10% of requests and 503 for another 5%
glitch --proxy https://api.staging.com --status 429:10,503:5
```

### 3. Limiting Bandwidth

Test connections by restricting response bandwidth. Instead of simply delaying the response, Glitch sends the payload in small bits at the set speed.

```bash
# Limit download speed to 50 kilobytes per second
glitch --proxy https://api.staging.com --bandwidth 50kbps

# Roughly like dial-up
glitch --proxy https://api.staging.com --bandwidth 5kb/s
```

### 4. Changing Data (Schema Resilience)

APIs do not always give your client what it expects. Glitch can change JSON responses so you can see how your frontend reacts when fields are gone, values become null, types change, or the payload is not valid.

Set up corruption through a profile or the global settings:

```yaml
# glitch.yaml

corruption:
  rate: 15 # 15% of JSON response payloads
  strategies: # Optional: choose specific mutators
    - Drop_field # Drop a random field from objects
    - Swap_type # Change value types
    - Inject_null # Replace a value with null
    - Break_syntax # Produce malformed JSON
  multi: true # Apply more than one mutator at the same time
```

### 5. Real-time Chaos (WebSockets & SSE)

Glitch can also interfere with real-time traffic. Use it to reproduce delay, lost messages, unexpected breaks, and out-of-order delivery in WebSocket and Server-Sent Events streams.

```yaml
# glitch.yaml

realtime:
  latency:
    fixed: ""
  drop_rate: 10
  disconnect_rate: 5
  out_of_order: true
```

### 6. Changing Failures Over Time

Sometimes you do not want one fixed failure for the test. Chaos Monkey mode allows Glitch to change its behavior over time, which is useful for checking recovery from issues or poor service.

```yaml
# glitch.yaml

monkey:
  enabled:
  phases:
    - Duration: "2m"
      failure:
        rate: 0

    - Duration: "30s"
      failure:
        rate: 100

    - Duration: "1m"
      latency:
        fixed: "3s"
```

### 7. Sharing Chaos Settings

If your team regularly tests the same problems, save them as YAML settings and put them in the repository.

For example, `.glitch/profiles/flaky.yaml`:

```yaml
latency:
  distribution: ""
  min: "1s"
  max: "4s"

failure:
  rate: 30

statuses:
  - Code: 502
    rate: 15

stall:
  rate: 5
  mode: drop

corruption:
  rate: 10
```

Then start the settings by name:

```bash
glitch --proxy https://api.example.com --profile flaky
```

This makes it simple for everyone on the team—and your CI jobs—to create the same conditions.

### 8. Specific Route Chaos (Partial Problems)

Production systems do not usually have all problems at once. Often one endpoint or service gets slow while the rest is fine.

Glitch allows you to change the chaos settings for certain routes. When multiple rules match, the specific path wins.

```yaml
# glitch.yaml

failure:
  rate: 0 # Stable by default

routes:
  - Path: "/api/checkout"
    method: POST
    failure:
      rate: 50 # Half of checkout requests fail

  - Path: "/api/products/*"
    latency:
      fixed: "3s" # Product routes are always slow
```

---

## 🔌 Main Interceptor Modes

Glitch needs an API to sit in front of. Depending on what you're working with, there are three ways to provide one.

### Reverse Proxy (Recommended)

If you have a staging API already, let Glitch know:

```bash
glitch --proxy https://api.mycompany.staging.com
```

*Glitch deals with CORS issues and ignores self-signed TLS errors so local `localhost` apps and E2E tests can talk to APIs without extra work.*

### OpenAPI Mock Server

If you have an OpenAPI v3 specification, Glitch can make a mock server from it:

```bash
glitch api.yaml
```

*Responses are created from the data types and structures in the specification.*

### Simple JSON Database

No OpenAPI specification yet? A simple JSON file works too.

```json
{
  "users": [
    {
      "id": 1,
      "name": "Alice"
    }
  ]
}
```

Run:

```bash
glitch db.json
```

*Glitch offers the data as a REST CRUD API with `GET`, `POST`, `PUT`, `PATCH`, and `DELETE`, plus built-in sorting, filtering, and page display.*

---

## 🛠️ Configuration Settings

If you use the setup often, you do not need to use a long list of CLI flags each time.

Put the settings in `glitch.yaml` or `.glitch.yaml` in your working directory and Glitch finds it automatically.

```yaml
# glitch.yaml

port: 8080

proxy:

verbose: true

latency:
  distribution: "normal"
  min: 200ms
  max: 2s

failure:
  rate: 15

statuses:
  - Code: 502
    rate: 10

bandwidth: 50kbps

stall:
  rate: 5
  mode: drop
  drop_at: 50

corruption:
  rate: 10
  strategies:
    - Drop_field
    - Inject_null
  multi: false
```

Once the file is there, just run:

```bash
glitch
```

*CLI flags take priority over values in the global configuration, so you can change specific settings when needed.*

---

## Installation

### Using Go

With Go 1.20 or newer installed:

```bash
go install github.com/st1lson/glitch/cmd/glitch@latest
```

### Using Docker

Glitch is also available on GitHub Container Registry as an Alpine-based image.

```bash
docker run -p 3000:3000 --proxy https://api.staging.com --fail-rate 10
```

It also works well in a `docker-compose.yml` setup when you want Playwright or Cypress tests to run against a deliberately unreliable local backend.

---

## CLI Help

```text
Usage:
  glitch [file] [flags]

Flags:
      --bandwidth string   limit the speed of the response (for example: "50kbps", "1mbps")
      --config string      location of the config file (default: automatically finds glitch.yaml)
      --fail-rate string   total percentage of failures (for example: 20)
  -h, --help               help for glitch
      --host string        host to connect to (default "localhost")
      --latency string     add delay (for example: 2s, normal:500ms,2s, uniform:1s,3s)
      --no-tui             turn off the interactive dashboard and use regular output for logging
  -p, --port int           port to use (default 3000)
      --profile string     name of a chaos profile to use
      --proxy string       send requests to this target URL instead of using a local file
      --read-only          do not save changes to the JSON database
      --status strings     list of specific status failures separated by commas (for example: 500:10,429:5)
  -v, --verbose            turn on detailed logging (shows request and response bodies)
```