# Media Upload — Validated Images → R2/S3, Converted to AVIF

Image upload for a multi-tenant SaaS: strict validation, conversion to **AVIF** (multi-size, responsive), stored on **R2 or S3** (S3-compatible API), tenant-isolated end to end. The convert step is **asynchronous** (the request returns fast; a worker does the CPU-heavy encode), so this reference touches the API, a worker binary, the DB, storage, and the public site's `srcset`.

It reuses primitives the skill already defines: `store.InTenant` (RLS, `backend-golang.md`), the `httperr` envelope (`backend-golang.md`), asynq jobs (`backend-golang.md` → Background Jobs), RBAC permissions (`rbac.md`), and the two-layer public cache (`frontend-vue-cloudflare.md`).

---

## Flow

```
client → POST /api/dashboard/media (auth + RequirePermission "media:upload")
  → Go API streams the body (capped reader)
  → STRICT validate: sniff magic bytes → image.DecodeConfig (dims + pixel-bomb cap)
                     → size/plan limit → input format allowlist
  → store ORIGINAL bytes to PRIVATE bucket  (tenants/{tid}/originals/{uuid})
  → insert media row (status=processing) via store.InTenant   ← RLS
  → enqueue asynq job { media_id, tenant_id }
  → 202 Accepted { id, status: "processing" }

worker (cmd/worker) → pull original from private bucket
  → vips CLI shell-out → AVIF at N sizes (thumb / medium / full), metadata stripped
  → upload each variant to PUBLIC bucket (tenants/{tid}/media/{media_id}/{variant}.avif)
  → update row: status=ready, variants JSONB   ← store.InTenant (tenant from job payload)
  (on failure → status=failed, error logged with correlation id; originals are kept permanently — see backup-recovery.md)

serve → row.variants → public variant URLs → SSR <img srcset> (ui-public.md)
```

> **The DB row is the source of truth and the access-control point, not the object.** A `media` row (RLS-protected) records which tenant owns the image and whether it's `ready`. The bucket objects are just bytes under tenant-scoped, random keys. Never derive ownership from the object key alone.

---

## Folder Structure

```
cmd/
  api/main.go
  worker/main.go        ← NEW: asynq worker process (runs the AVIF conversion)
internal/
  media/
    handler.go          ← HTTP: receive upload, validate, store original, enqueue
    service.go          ← orchestration via store.InTenant
    validator.go        ← strict byte-level validation (the security core)
    converter.go        ← vips CLI shell-out → AVIF variants
    job.go              ← asynq task type + worker handler
    types.go            ← Media, Variant, size presets
  storage/
    storage.go          ← Storage interface (Put/Get/Delete/PublicURL)
    s3.go               ← one S3-compatible impl: R2 AND S3 (differ only by endpoint)
query/
  media.sql             ← sqlc queries (CreateMedia, MarkReady, MarkFailed, GetMedia, ...)
migrations/
  00NN_create_media.up.sql / .down.sql
```

Reused, not re-created: `internal/store` (RLS helper), `internal/httperr`, `internal/auth` (RBAC), `pkg/cache` (asynq client lives alongside Redis).

---

## Database: `media` table (RLS, same checklist as every tenant table)

```sql
-- migrations/00NN_create_media.up.sql
CREATE TABLE media (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status       VARCHAR(20) NOT NULL DEFAULT 'processing', -- processing | ready | failed
    original_key TEXT NOT NULL,                  -- key in the PRIVATE bucket
    mime_in      VARCHAR(50) NOT NULL,            -- validated source type (image/jpeg, ...)
    width        INT NOT NULL,                    -- of the original, from DecodeConfig
    height       INT NOT NULL,
    bytes_in     BIGINT NOT NULL,                 -- original size
    variants     JSONB NOT NULL DEFAULT '[]',     -- [{ name, key, w, h, bytes }] once ready
    error        TEXT,                            -- failure reason (server-side detail)
    created_by   UUID REFERENCES tenant_users(id),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE media ENABLE ROW LEVEL SECURITY;
ALTER TABLE media FORCE  ROW LEVEL SECURITY;   -- required: close the table-owner gap

CREATE POLICY tenant_isolation ON media
    USING       (tenant_id = current_tenant_id())
    WITH CHECK  (tenant_id = current_tenant_id());

CREATE INDEX idx_media_tenant_status  ON media(tenant_id, status);
CREATE INDEX idx_media_tenant_created ON media(tenant_id, created_at DESC);
```

Follows the new-table checklist in `database-schema.md` exactly: `tenant_id NOT NULL`, ENABLE+FORCE RLS, USING+WITH CHECK with `current_tenant_id()`, composite index `tenant_id`-first. Add a row to the isolation test for it.

---

## Strict Validation — the security core (`validator.go`)

> **Never trust the client.** `Content-Type` headers and file extensions are attacker-controlled. Validation must inspect the actual bytes, and the *real* sanitizer is re-encoding (below) — a malicious payload doesn't survive a decode→re-encode. These checks run BEFORE the original is stored.

```go
package media

import (
    "bytes"
    "errors"
    "image"
    _ "image/jpeg" // register decoders for DecodeConfig
    _ "image/png"
    "net/http"

    _ "golang.org/x/image/webp" // webp DecodeConfig
)

// Limits — tune per plan (free vs pro). Pull from config; constants here for clarity.
const (
    maxBytes      = 15 << 20 // 15 MB hard cap on the request body
    maxPixels     = 40_000_000 // 40 MP — anti decompression-bomb (e.g. 8000x5000)
    maxDimension  = 12_000     // no single side larger than this
)

// allowedInput — source formats we accept. Output is ALWAYS AVIF regardless.
var allowedInput = map[string]bool{
    "image/jpeg": true, "image/png": true, "image/webp": true,
    // HEIC is common from phones; decode happens in vips at convert time, but we still
    // sniff+gate it here. If you don't ship a HEIC sniffer, drop it from the allowlist.
}

type Validated struct {
    Bytes  []byte
    Mime   string
    Width  int
    Height int
}

// Validate runs on the in-memory bytes (already capped by a limited reader, see handler).
func Validate(raw []byte) (*Validated, *httperr.AppError) {
    // 1. Sniff real type from magic bytes — NOT the Content-Type header.
    mime := http.DetectContentType(raw)
    if !allowedInput[mime] {
        return nil, httperr.UnsupportedMedia("Unsupported image type")
    }

    // 2. Decode just the header to get dimensions WITHOUT allocating the full pixel buffer.
    //    This is the decompression-bomb guard: reject huge dims before any heavy decode.
    cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
    if err != nil {
        return nil, httperr.Validation(map[string]string{"file": "not a valid image"})
    }
    if cfg.Width > maxDimension || cfg.Height > maxDimension {
        return nil, httperr.Validation(map[string]string{"file": "image dimensions too large"})
    }
    if cfg.Width*cfg.Height > maxPixels {
        return nil, httperr.Validation(map[string]string{"file": "image has too many pixels"})
    }

    return &Validated{Bytes: raw, Mime: mime, Width: cfg.Width, Height: cfg.Height}, nil
}
```

```go
// handler.go — cap the body BEFORE reading it into memory (don't trust Content-Length).
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, maxBytes) // 413 if exceeded
    raw, err := io.ReadAll(r.Body)
    if err != nil {
        httperr.Write(w, httperr.PayloadTooLarge("File exceeds the size limit"))
        return
    }

    v, appErr := Validate(raw)
    if appErr != nil {
        httperr.Write(w, appErr) // 415 or 422 with field message
        return
    }

    media, appErr := h.svc.CreateAndEnqueue(r.Context(), v) // stores original + row + job
    if appErr != nil {
        httperr.Write(w, appErr)
        return
    }
    w.WriteHeader(http.StatusAccepted) // 202 — conversion happens async
    json.NewEncoder(w).Encode(media)   // { id, status: "processing" }
}
```

Validation summary (all enforced before anything is stored):
- **Magic-byte sniff** (`http.DetectContentType`) — not header/extension.
- **Dimension + pixel-count caps** via `DecodeConfig` (header-only) — anti decompression-bomb.
- **Size cap** via `http.MaxBytesReader` — doesn't trust `Content-Length`.
- **Input allowlist** → output always AVIF; re-encode strips EXIF/GPS/hidden payloads.

---

## Storage — one S3-compatible client for R2 and S3 (`storage.go` / `s3.go`)

```go
package storage

import "context"

// Storage abstracts object storage. R2 and S3 differ ONLY by endpoint/region config,
// so one aws-sdk-go-v2 S3 client serves both — swap via env, no code change.
type Storage interface {
    Put(ctx context.Context, bucket, key string, body []byte, contentType string) error
    Get(ctx context.Context, bucket, key string) ([]byte, error)
    Delete(ctx context.Context, bucket, key string) error
    PublicURL(key string) string // variant URL on the public bucket's custom domain
}
```

```go
// s3.go — aws-sdk-go-v2 with a custom endpoint resolver = works for R2 and S3.
func NewS3(cfg Config) (*S3, error) {
    awsCfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(cfg.Region), // R2: "auto"
        config.WithCredentialsProvider(creds(cfg.AccessKey, cfg.SecretKey)),
    )
    if err != nil { return nil, err }
    client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String(cfg.Endpoint) // R2: https://<acct>.r2.cloudflarestorage.com
        o.UsePathStyle = cfg.PathStyle            // true for some S3-compatibles
    })
    return &S3{client: client, publicBase: cfg.PublicBaseURL}, nil
}
```

> **Two buckets, two visibility levels:**
> - **Originals → PRIVATE bucket.** Raw uploads are never publicly reachable. They exist for re-processing and audit, and are **kept permanently with object versioning** — they are the authoritative copy and a backup artifact (lets you re-encode new variant sizes/quality later, or recover a lost variant). See `backup-recovery.md`.
> - **Variants → PUBLIC bucket**, served via a custom domain / CDN. Only the re-encoded, metadata-stripped AVIFs are exposed; they are derived (regenerable from the original), so not separately backed up.

### Object keys — tenant-scoped, random, never the client filename

```go
// Tenant prefix isolates blobs per tenant; UUIDs prevent guessing, collisions, overwrite,
// and path traversal. The client's filename is NEVER used in a key.
func originalKey(tenantID string) string {
    return fmt.Sprintf("tenants/%s/originals/%s", tenantID, uuid.NewString())
}
func variantKey(tenantID, mediaID, variant string) string {
    return fmt.Sprintf("tenants/%s/media/%s/%s.avif", tenantID, mediaID, variant)
}
```

---

## Service — store original + row + job, all RLS-bound (`service.go`)

```go
func (s *Service) CreateAndEnqueue(ctx context.Context, v *Validated) (*db.Media, *httperr.AppError) {
    t := tenant.MustFromContext(ctx)
    key := originalKey(t.ID)

    // 1. Store the validated ORIGINAL to the private bucket.
    if err := s.storage.Put(ctx, s.cfg.PrivateBucket, key, v.Bytes, v.Mime); err != nil {
        return nil, httperr.Internal("Could not store upload")
    }

    // 2. Insert the row (status=processing) inside an RLS tx — tenant_id from current_tenant_id().
    var m db.Media
    err := s.store.InTenant(ctx, func(q *db.Queries) error {
        var e error
        m, e = q.CreateMedia(ctx, db.CreateMediaParams{
            OriginalKey: key, MimeIn: v.Mime,
            Width: int32(v.Width), Height: int32(v.Height), BytesIn: int64(len(v.Bytes)),
            CreatedBy: userIDFromCtx(ctx),
        })
        return e
    })
    if err != nil {
        _ = s.storage.Delete(ctx, s.cfg.PrivateBucket, key) // best-effort orphan cleanup
        return nil, httperr.Internal("Could not record upload")
    }

    // 3. Enqueue the convert job. tenant_id travels in the payload — the worker has no
    //    request context, so it re-establishes tenant via store.InTenant from this value.
    if err := s.jobs.EnqueueConvert(ctx, ConvertPayload{MediaID: m.ID, TenantID: t.ID}); err != nil {
        // Row stays 'processing'; a sweeper re-enqueues stuck rows. Don't fail the user here.
        s.log.Error("enqueue convert failed", "media_id", m.ID, "err", err)
    }
    return &m, nil
}
```

---

## Async Conversion — vips CLI shell-out (`converter.go`, `job.go`)

> **Why CLI, not cgo.** AVIF needs libvips/libaom; there's no mature pure-Go encoder. Shelling out to the `vips` binary keeps the Go build `CGO_ENABLED=0` (static, as `deployment.md` assumes) and isolates the heavy native work in a subprocess we can timeout/kill. The trade-off is an OS dependency (`vips` installed on the worker host — see `deployment.md`).

```go
// Size presets → responsive srcset. Downscale only (never upscale past the original).
var sizes = []struct {
    Name string
    W    int
}{
    {"thumb", 320}, {"medium", 768}, {"full", 1600},
}

// convert runs vips per size. Security notes:
//  - exec.Command with explicit args (NO shell string) → no command injection.
//  - work on temp files with a hard timeout; kill on overrun.
//  - vips re-encode DROPS metadata by default here (strip) → no EXIF/GPS leak.
func convertOne(ctx context.Context, srcPath, dstPath string, width int) error {
    ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
    defer cancel()

    // thumbnail = downscale with aspect preserved; avifsave with quality + strip metadata.
    // `[Q=50,strip=true]` → quality 50 AVIF, no metadata. width<=0 keeps native (full).
    args := []string{
        "thumbnail", srcPath,
        dstPath + "[Q=50,strip=true]", // avifsave options on the output target
        strconv.Itoa(width),
        "--size", "down", // never enlarge
    }
    cmd := exec.CommandContext(ctx, "vips", args...)
    var stderr bytes.Buffer
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("vips: %w: %s", err, stderr.String())
    }
    return nil
}
```

```go
// job.go — the worker handler. Runs in cmd/worker, no HTTP context.
func (w *Worker) HandleConvert(ctx context.Context, p ConvertPayload) error {
    // Re-establish tenant context from the payload so store.InTenant + RLS apply.
    ctx = tenant.WithTenant(ctx, &tenant.Tenant{ID: p.TenantID})

    var m db.Media
    if err := w.store.InTenant(ctx, func(q *db.Queries) error {
        var e error; m, e = q.GetMedia(ctx, p.MediaID); return e
    }); err != nil {
        return err // asynq retries; if the row is gone, the retry no-ops
    }

    orig, err := w.storage.Get(ctx, w.cfg.PrivateBucket, m.OriginalKey)
    if err != nil { return w.fail(ctx, p, "fetch original", err) }

    tmp := writeTemp(orig) // temp file for vips input
    defer os.Remove(tmp)

    variants := make([]Variant, 0, len(sizes))
    for _, sz := range sizes {
        if sz.W > int(m.Width) && sz.Name != "full" { continue } // skip upscales
        out := tmp + "." + sz.Name + ".avif"
        if err := convertOne(ctx, tmp, out, sz.W); err != nil {
            return w.fail(ctx, p, "convert "+sz.Name, err)
        }
        defer os.Remove(out)
        body, _ := os.ReadFile(out)
        key := variantKey(p.TenantID, p.MediaID.String(), sz.Name)
        if err := w.storage.Put(ctx, w.cfg.PublicBucket, key, body, "image/avif"); err != nil {
            return w.fail(ctx, p, "upload "+sz.Name, err)
        }
        variants = append(variants, Variant{Name: sz.Name, Key: key, W: sz.W, Bytes: len(body)})
    }

    // Mark ready inside an RLS tx.
    return w.store.InTenant(ctx, func(q *db.Queries) error {
        return q.MarkReady(ctx, db.MarkReadyParams{ID: p.MediaID, Variants: toJSON(variants)})
    })
}

// fail records status=failed (detail server-side) and returns nil so asynq won't retry
// a deterministic failure (e.g. corrupt source). Transient errors return the error to retry.
func (w *Worker) fail(ctx context.Context, p ConvertPayload, stage string, err error) error {
    w.log.Error("convert failed", "stage", stage, "media_id", p.MediaID, "err", err)
    _ = w.store.InTenant(ctx, func(q *db.Queries) error {
        return q.MarkFailed(ctx, db.MarkFailedParams{ID: p.MediaID, Error: stage + ": " + err.Error()})
    })
    return nil
}
```

---

## Authorization

Gate the routes with the tenant permission model (`rbac.md`):

```go
r.With(RequirePermission("media:upload")).Post("/api/dashboard/media", mediaHandler.Upload)
r.With(RequirePermission("media:read")).Get("/api/dashboard/media", mediaHandler.List)
r.With(RequirePermission("media:delete")).Delete("/api/dashboard/media/{id}", mediaHandler.Delete)
```

`media:upload`, `media:read`, `media:delete` join the permission format in `rbac.md`. Delete removes the row (RLS-scoped) and both bucket objects.

---

## Serving — variant URLs feed the public `srcset`

The `ready` row's `variants` map to public URLs. The public SSR site (`ui-public.md`) renders a responsive `<img>`; the variant objects are immutable and cacheable by the two-layer public cache.

```vue
<!-- in a public content component (ui-public.md) -->
<img
  :src="m.variants.full.url"
  :srcset="`${m.variants.thumb.url} 320w, ${m.variants.medium.url} 768w, ${m.variants.full.url} 1600w`"
  sizes="(max-width: 768px) 100vw, 768px"
  :width="m.variants.full.w" :height="origHeightFor('full', m)"
  loading="lazy" decoding="async" alt=""
/>
```

> Variant URLs are stable and content-addressed by `media_id` + variant, so they cache indefinitely at the edge. If an image is replaced, it gets a new `media_id` (new keys) — no purge needed; the old URLs simply stop being referenced.

---

## Security Checklist

- [ ] Validate by **magic bytes + decode**, never `Content-Type`/extension.
- [ ] **Pixel-count + dimension caps** before heavy decode (decompression-bomb).
- [ ] **Body size cap** via `MaxBytesReader` (don't trust `Content-Length`).
- [ ] Re-encode to AVIF **strips metadata** (EXIF/GPS) — set `strip=true`.
- [ ] Object keys are **tenant-scoped + random UUID**; client filename never used.
- [ ] `vips` invoked with **explicit args** (`exec.Command`, no shell) + **timeout**.
- [ ] Originals **private bucket**; only stripped AVIF variants are public.
- [ ] `media` row is **RLS-protected** (ENABLE+FORCE, USING+WITH CHECK); worker re-binds tenant from the job payload via `store.InTenant`.
- [ ] Errors use the `httperr` envelope (413/415/422); failures logged server-side, not leaked.
