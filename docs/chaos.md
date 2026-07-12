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
