# Chaos Engineering Guide

Glitch is primarily designed to inject chaos into your frontend stack. You can trigger chaos flags via the CLI or define them in your `glitch.yaml` configuration.

## Latency Injection

Testing loading states and skeleton UIs often requires throttling the entire browser in Chrome DevTools. Glitch allows you to target latency precisely at the API layer.

### Fixed Latency
Adds a flat delay to every single request.
```bash
glitch --proxy https://api.staging.com --latency 2s
```

### Variable Latency
Sometimes the network is just unpredictable. You can simulate varying latency across a range using different probability distributions.

**Uniform Distribution**: Every value between min and max has an equal chance of occurring.
```bash
glitch --proxy https://api.staging.com --latency uniform:500ms,3s
```

**Normal Distribution**: A bell curve. Most requests will fall in the middle of the range, with rare outliers at the extremes.
```bash
glitch --proxy https://api.staging.com --latency normal:200ms,2s
```

---

## Failure Injection

Stop hardcoding `throw new Error()` into your React components. Let Glitch break things for you.

### Random Failure Rate
To simulate a flaky backend, you can specify an overall failure rate percentage. Glitch will drop requests randomly.
```bash
glitch --proxy https://api.staging.com --fail-rate 20
```

### Specific Status Codes
If you want to test how your frontend handles specific scenarios—like a `429 Too Many Requests` or a `503 Service Unavailable`—you can specify the exact status codes and how often they should occur.
```bash
# 10% of requests return 429, 5% of requests return 503
glitch --proxy https://api.staging.com --status 429:10,503:5
```

---

## Bandwidth Throttling 🐢

If you have large JSON payloads or large media files, adding latency isn't enough. A 2-second latency delay will pause for 2 seconds, and then instantly deliver the 5MB payload. 

Bandwidth throttling drips the response to the client in chunks, perfectly simulating slow cellular networks.

```bash
# Throttle downloads to 50 Kilobytes per second
glitch --proxy https://api.staging.com --bandwidth 50kbps

# Dial-up speeds
glitch --proxy https://api.staging.com --bandwidth 5kb/s
```

---

## Stall Injection (Mid-Flight Aborts)

Sometimes connections don't fail immediately. Often, particularly on mobile devices switching networks, a request is accepted, some data is downloaded, and then it simply hangs forever or suddenly drops mid-flight (TCP reset).

Stall Injection lets you simulate this behavior by streaming a configurable percentage of the payload before abruptly hanging or dropping the connection.

*Note: Stall Injection is currently configured via your `glitch.yaml` file.*

```yaml
stall:
  rate: 5        # 5% chance of the connection stalling
  mode: drop     # "drop" (TCP reset) or "hang" (block indefinitely)
  drop_at: 50    # Stream 50% of the payload before stalling
```

---

## Payload Corruption (Schema Resilience)

It is common for frontends or API clients to crash when the server responds with a payload that doesn't match the expected schema (e.g., missing fields, unexpected types, or null values).

Payload Corruption allows you to simulate these backend schema deviations dynamically. When enabled, Glitch buffers JSON response payloads and applies random mutations to the structure or data before flushing it to the client.

*Note: Payload Corruption is currently configured via your `glitch.yaml` file or a chaos profile.*

### Configuration

```yaml
corruption:
  rate: 10              # 10% chance of mutating a JSON response
  strategies:           # (Optional) List of active mutators. If omitted, all are active.
    - drop_field
    - swap_type
    - inject_null
    - break_syntax
  multi: true           # (Optional) If true, applies 2 to 4 mutators per response instead of just 1
```

### Mutation Strategies

Glitch supports four built-in mutation strategies:

1. **`drop_field`**:
   - For JSON objects: Removes a random key-value pair from the object.
   - For JSON arrays: Removes a random element from the array.
   
2. **`swap_type`**:
   - Changes the type of a value at a random depth in the JSON payload:
     - `string` values are replaced by an integer (`42`).
     - `number` values (int/float64) are replaced by a string (`"corrupted_string"`).
     - `boolean` values are replaced by an integer (`1`).
     - `null` values are replaced by a string (`"not_null"`).
   
3. **`inject_null`**:
   - Replaces a random value (field value or array element) with `null`.
   
4. **`break_syntax`**:
   - Directly corrupts the raw JSON byte buffer:
     - **Truncation**: Cuts the JSON string in half, returning invalid/incomplete JSON.
     - **Trailing Comma**: Injects a trailing comma before the closing bracket `]` or brace `}`.
     - **Unescaped Quote**: Injects an unescaped double quote `"` in the middle of the JSON string.

---

## Real-time Chaos (WebSockets & SSE)

Testing real-time applications is notoriously difficult. Glitch allows you to inject chaos directly into Server-Sent Events (SSE) and WebSocket streams.

*Note: Real-time chaos is currently configured via your `glitch.yaml` file or a chaos profile.*

### Configuration

```yaml
realtime:
  latency:
    fixed: "500ms"          # Add 500ms of latency to each outgoing message
  drop_rate: 10             # 10% chance of a message being silently dropped
  disconnect_rate: 5        # 5% chance the connection is abruptly closed
  out_of_order: true        # Randomize the delivery order of messages
  max_buffered_messages: 50 # (Optional) How many messages to buffer for out-of-order delivery. Default 100.
```

### Chaos Features

- **`latency`**: Delays the delivery of individual messages. Supports `fixed`, `min`/`max` ranges, and distributions just like the global latency config.
- **`drop_rate`**: Simulates packet loss by silently dropping messages before they reach the client.
- **`disconnect_rate`**: Tests reconnection logic by randomly dropping the entire connection while a stream is active.
- **`out_of_order`**: Buffers messages and flushes them in a randomized order to simulate race conditions or out-of-order packet delivery.

---

## Chaos Monkey Mode

Instead of static failure rates and latency, you can configure Glitch to dynamically cycle between different chaos phases over time. This is incredibly useful for testing how your frontend applications recover from network outages or degraded states (e.g., reconnecting to a WebSocket, backoff polling, etc.).

*Note: Chaos Monkey Mode is currently configured via your `glitch.yaml` file.*

```yaml
monkey:
  enabled: true
  phases:
    - duration: "2m"       # Phase 1: API works perfectly for 2 minutes
      failure:
        rate: 0
    - duration: "30s"      # Phase 2: API goes completely offline for 30s
      failure:
        rate: 100
    - duration: "1m"       # Phase 3: API recovers, but is very slow for 1 min
      latency:
        fixed: "3s"
      failure:
        rate: 0

---

## Route-Specific Chaos (Partial Degradation)

Real production environments rarely go down entirely; usually, just one microservice (like search or payments) degrades. You can override global chaos settings for specific API endpoints. The most specific path match wins.

*Note: Route-specific chaos is configured via your `glitch.yaml` file.*

### Specificity Scoring
When multiple routes match an incoming request, Glitch selects the most specific one:
- **Exact Path Matches** win over wildcard matches.
- **Longer Wildcard Matches** (e.g. `/api/products/*`) win over shorter wildcard matches (e.g. `*`).
- **Method-Specific Overrides** (e.g. `method: POST`) win over method-agnostic overrides for the same path.

### Configuration Example

```yaml
# glitch.yaml
failure:
  rate: 0 # Global baseline is stable

routes:
  - path: "/api/checkout"
    method: POST
    failure:
      rate: 50 # But POSTs to checkout fail 50% of the time
  - path: "/api/products/*"
    latency:
      fixed: "3s" # All product routes are extremely slow
```
```
