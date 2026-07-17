# Global Configuration

While the CLI is great for quick tests, passing `--latency normal:200ms,2s --fail-rate 15 --status 502:10` every time you start your local environment is exhausting.

Glitch natively supports YAML configurations. It automatically auto-discovers `glitch.yaml` or `.glitch.yaml` in your current working directory.

## glitch.yaml

```yaml
# Server Settings
port: 8080
host: localhost

# Interceptor target
proxy: http://api.staging.internal
# file: db.json  <-- use this instead of proxy if mocking

# Base Chaos Settings
latency:
  distribution: "normal"
  min: 200ms
  max: 2s

failure:
  rate: 15
  statuses:
    - code: 502
      rate: 10
    - code: 429
      rate: 5

bandwidth: 50kbps

stall:
  rate: 5
  mode: drop
  drop_at: 50

# Payload corruption parameters
corruption:
  rate: 10
  strategies:
    - drop_field
    - swap_type
  multi: true

# Route-Specific Overrides
routes:
  - path: "/api/checkout"
    method: POST
    failure:
      rate: 50
  - path: "/api/products/*"
    latency:
      fixed: "3s"
```

Once this file exists, you can just type:
```bash
glitch
```

*Note: CLI flags always take precedence over the global config file, so you can easily override settings on the fly.*

## Chaos Profiles

You can store specific scenarios in a `.glitch/profiles/` directory and apply them dynamically. This is a fantastic way to align your QA team on specific testing conditions.

Create `.glitch/profiles/mobile-3g.yaml`:
```yaml
latency:
  distribution: "uniform"
  min: 1s
  max: 3s
bandwidth: 150kbps
```

Create `.glitch/profiles/outage.yaml`:
```yaml
failure:
  rate: 100
  statuses:
    - code: 503
      rate: 100
stall:
  rate: 100
  mode: drop
  drop_at: 10
corruption:
  rate: 50
  multi: true
```

To run a profile, simply pass the `--profile` flag:
```bash
glitch --profile mobile-3g
```

If you are using the interactive **Terminal UI (TUI)**, you can press the **left (←)** and **right (→)** arrow keys to dynamically cycle between your defined chaos profiles while the server is running without restarting!
