# API Contracts & Versioning

API contracts are the boundary between Go backend, Cloudflare Workers, Vue dashboard, public SSR, tests, and future integrations. The backend is authoritative; frontend and Worker code must consume the documented contract rather than infer behavior from implementation details.

---

## Canonical Contract Rules

```text
[ ] API routes are versioned or explicitly marked internal
[ ] All API errors use the standard envelope
[ ] Error codes are stable and documented
[ ] Validation errors map to fields predictably
[ ] Pagination/filter/sort formats are consistent
[ ] Auth/tenant boundaries are explicit per route
[ ] OpenAPI or equivalent schema is generated/maintained
[ ] Frontend clients are typed from the contract where practical
[ ] Contract tests run in CI
```

Canonical error envelope from `backend-golang.md`:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Please check the highlighted fields.",
    "fields": {
      "email": "Email is required"
    }
  }
}
```

Workers that synthesize API JSON errors must use the same envelope. HTML navigations may return redirects or HTML error pages; `/api/*` and XHR/fetch responses must not.

---

## URL Versioning

Recommended public contract shape:

```text
/api/v1/public/*       anonymous tenant public reads
/api/v1/dashboard/*    authenticated tenant dashboard
/api/v1/platform/*     authenticated platform admin
/api/v1/auth/*         login/refresh/logout/password reset
/api/v1/webhooks/*     provider webhooks; signature-authenticated
/api/internal/*        network-isolated Worker/backend internal APIs, not public contract
```

If existing examples omit `/v1`, treat that as shorthand. New production implementations should choose one convention and apply it consistently. Do not mix versioned and unversioned route trees without explicit redirects/compatibility wrappers.

Rules:

- Breaking response changes require a new version or compatibility window.
- Additive fields are allowed within the same version.
- Removing/renaming fields is breaking.
- Changing enum values is breaking unless documented as open-ended.
- Error `code` changes are breaking for clients that branch on them.

---

## Route Contract Template

Every route should declare:

```markdown
## <METHOD> /api/v1/dashboard/example

Auth: tenant user | platform user | public | internal | provider webhook
Tenant context: required | optional | none
Permission: example:read
Rate limit: rule name from rate-limiting.md
Audit: event name if sensitive
Idempotency: required? header/body key?
Request body/query:
Response 2xx:
Errors:
Caching: no-store | public cacheable | private
Metrics route label: /api/v1/dashboard/example
```

Example:

```markdown
## POST /api/v1/dashboard/media

Auth: tenant user
Tenant context: required, from JWT tenant_id cross-checked with X-Tenant-ID hint
Permission: media:upload
Rate limit: media_upload_user
Audit: media.uploaded
Idempotency: optional `Idempotency-Key` for retrying upload metadata creation
Request: multipart/form-data, max size by plan
Response 201:
  { "data": { "id": "uuid", "status": "processing" } }
Errors: unauthorized, forbidden, validation_failed, too_many_requests, payload_too_large
Caching: no-store
Metrics route label: /api/v1/dashboard/media
```

---

## Success Response Shape

Use a predictable wrapper for JSON API responses:

```json
{
  "data": {
    "id": "..."
  }
}
```

List response:

```json
{
  "data": [
    { "id": "..." }
  ],
  "page": {
    "limit": 20,
    "next_cursor": "opaque-cursor-or-null"
  }
}
```

Do not return top-level arrays for API endpoints; wrappers allow pagination/meta additions without breaking clients.

---

## Pagination, Filtering, Sorting

Default to cursor pagination for mutable lists:

```text
GET /api/v1/dashboard/audit?limit=50&cursor=opaque
```

Rules:

- `limit` has server max.
- `cursor` is opaque to clients.
- Sort keys are allowlisted.
- Filters are allowlisted and validated.
- Never expose raw SQL column names as public API contract unless intentionally stable.

Response:

```json
{
  "data": [],
  "page": {
    "limit": 50,
    "next_cursor": null
  }
}
```

---

## Error Code Registry

Stable baseline codes:

```text
unauthorized
forbidden
not_found
validation_failed
too_many_requests
payload_too_large
conflict
idempotency_conflict
csrf_failed
invalid_signature
expired_code
invalid_code
plan_limit_exceeded
tenant_suspended
dependency_unavailable
internal_error
```

Field validation error:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Please check the highlighted fields.",
    "fields": {
      "name": "Name is required",
      "email": "Email must be valid"
    }
  }
}
```

500 errors:

```json
{
  "error": {
    "code": "internal_error",
    "message": "Something went wrong. Reference: req_abc123"
  }
}
```

Full internal details are logged server-side with `request_id`, never returned.

---

## Idempotency

Required for operations where retries can duplicate side effects:

```text
billing charge creation
payment webhook processing
tenant provisioning jobs
email send jobs
media metadata creation if client retries uploads
```

Client-facing idempotency header:

```text
Idempotency-Key: <opaque-client-generated-key>
```

Rules:

- Scope by tenant/user/operation.
- Store durable result for replay where necessary.
- Return same successful response for same key/body.
- Return `409 idempotency_conflict` if same key has different body.
- Provider webhooks use provider event/message IDs, not client header.

---

## OpenAPI

Recommended:

```text
api/openapi.yaml
```

Generate or validate in CI:

```bash
npx @redocly/cli lint api/openapi.yaml
npx openapi-typescript api/openapi.yaml -o vue-dashboard/src/api/schema.ts
```

Go route handlers should either be generated from the schema or tested against it. If full generation is too heavy, keep OpenAPI as the published contract and add contract tests for critical routes.

---

## Typed Frontend Client

The Vue dashboard should use one API client module:

```text
src/utils/api.ts          → fetch wrapper, envelope parsing
src/api/schema.ts         → generated OpenAPI types
src/api/dashboard.ts      → typed endpoint functions
```

Rules:

- Components do not call `fetch` directly.
- `ApiError` parses `{ error: { code, message, fields? } }`.
- `422 validation_failed` maps to base form components (`ui-dashboard.md`).
- `401` triggers session-ended handling only after Worker silent refresh has failed.

---

## Backward Compatibility Policy

Additive changes allowed:

```text
new optional response field
new optional request field
new enum value only if clients treat enum as open-ended
new endpoint
new error code for a newly documented case
```

Breaking changes:

```text
remove/rename response field
change field type
change error code semantics
change pagination format
change auth/permission requirement without release note
change cacheability of endpoint
```

Breaking changes require:

```text
[ ] new API version or compatibility shim
[ ] release note
[ ] frontend update
[ ] contract test update
[ ] deprecation window if external clients exist
```

---

## Contract Tests

Required tests:

- All API errors use envelope.
- `422` field errors are under `error.fields`.
- Cross-tenant resource misses return `404`.
- Route-level permission failure returns `403`.
- Public cacheable endpoints never require auth and never return private data.
- Pagination response includes `data` and `page`.
- OpenAPI examples validate against schema.
- Vue API client handles success, validation error, unauthorized, rate limit, and 500.

Example:

```go
func AssertErrorEnvelope(t *testing.T, body []byte) {
    var got struct {
        Error struct {
            Code    string            `json:"code"`
            Message string            `json:"message"`
            Fields  map[string]string `json:"fields,omitempty"`
        } `json:"error"`
    }
    require.NoError(t, json.Unmarshal(body, &got))
    require.NotEmpty(t, got.Error.Code)
    require.NotEmpty(t, got.Error.Message)
}
```

---

## Definition of Done

```text
[ ] Route contract documented
[ ] Auth/tenant/permission/rate-limit/audit declared
[ ] Success response uses data/page wrapper
[ ] Error responses use canonical envelope
[ ] Error code added to registry if new
[ ] OpenAPI updated or contract test added
[ ] Vue typed client updated
[ ] Backward compatibility reviewed
[ ] Contract tests pass in CI
```
