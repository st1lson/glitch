# Interceptor Engines

Glitch isn't just a proxy; it dynamically adapts to your backend setup. It has three primary engines it uses to supply data to your frontend.

## 1. Reverse Proxy (Recommended)

If you have a live staging backend, or another local container running your real API, Glitch can sit in front of it and intercept traffic.

```bash
glitch --proxy https://api.staging.internal.company.com
```

**Features:**
- **CORS Bypass**: Glitch automatically intercepts `OPTIONS` preflight requests and strips restrictive CORS headers from the upstream, so your `http://localhost:5173` frontend can talk directly to remote production domains without browser warnings.
- **TLS Ignored**: Glitch ignores self-signed certificate errors on internal staging domains.

## 2. OpenAPI Mock Server

If your backend engineers haven't built the API yet, but they have provided a Swagger or OpenAPI v3 spec, Glitch can mock the entire API locally.

```bash
glitch api.yaml
```

**Features:**
- Parses `components/schemas` to auto-generate realistic fake data using `reggen` for regex-based patterns.
- Handles all paths dynamically, allowing your frontend to execute `GET`, `POST`, and `DELETE` requests with `200 OK` responses containing generated mock data.

## 3. Generic JSON Database

If you have absolutely nothing but a rough idea of what the JSON should look like, you can supply a raw `.json` file containing arrays of objects.

```json
{
  "users": [
    { "id": 1, "name": "Alice" },
    { "id": 2, "name": "Bob" }
  ],
  "posts": []
}
```

```bash
glitch db.json
```

**Features:**
- **Instant CRUD:** You instantly get `GET /users`, `POST /users`, `PUT /users/1`, `PATCH /users/1`, and `DELETE /users/1` endpoints.
- **Pagination:** Supports standard pagination via `?_page=1&_limit=10`.
- **Sorting:** Sort datasets via `?_sort=views&_order=desc`.
- **Persistence:** Any changes made via `POST`/`PUT` are written back to the `db.json` file on disk. (Disable this using the `--read-only` flag).
