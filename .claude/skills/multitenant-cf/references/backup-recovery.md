# Backup & Recovery — PITR, Per-Tenant Export, Object Storage

What to back up follows one rule: **authoritative state must be backed up; derived/ephemeral state is rebuilt, not backed up.** Get the boundary right and the plan is small; get it wrong and you either lose data or waste effort backing up things you can regenerate.

| Store | Role | Backup approach |
|---|---|---|
| **PostgreSQL** | **Source of truth** — tenants, users, roles, content, media metadata, `audit_log` | **PITR + nightly logical dump** (this doc) |
| **R2/S3 originals** | Partially authoritative — the only copy of raw uploaded bytes | **Keep permanently + object versioning** |
| R2/S3 variants (AVIF) | Derived from originals | Not backed up — regenerable by re-encoding |
| Redis | Cache / locks / jobs / refresh-family | Ephemeral — not backed up (loss = users re-login, caches refill) |
| Cloudflare KV | tenant config (derived from PG) + sessions (ephemeral) | Not backed up — rebuilt from PG; sessions expire anyway |

> **The boundary is "can I recompute this from PostgreSQL?"** KV tenant config is written from PG on tenant changes, so it's rebuildable. AVIF variants are re-encoded from originals. Sessions are short-lived by design. Only PG (and the raw originals it points at) hold state that cannot be reconstructed — so only those are backed up.

---

## PostgreSQL — two complementary layers

### 1. PITR (Point-In-Time Recovery) — the DR backbone

WAL archiving + periodic base backups let you restore to *any* moment (RPO ~minutes), which is what you need after a bad migration, a crash, or data corruption. Use **pgBackRest** (or WAL-G).

```ini
# /etc/pgbackrest/pgbackrest.conf
[global]
repo1-path=/var/lib/pgbackrest          # local staging; repo1-* below ships it off-box
repo1-retention-full=4                   # keep 4 full backups
repo1-cipher-type=aes-256-cbc            # encrypt at rest (REQUIRED — see Security)
repo1-cipher-pass=<from-secrets-manager>
# Ship the repo OFF the VPS — S3/R2/B2. A backup on the same box dies with the box.
repo1-s3-bucket=yourapp-pgbackup
repo1-s3-endpoint=<s3-or-r2-endpoint>
repo1-s3-key=<backup-only-key>
repo1-s3-key-secret=<backup-only-secret>

[yourdb]
pg1-path=/var/lib/postgresql/16/main
```

```ini
# postgresql.conf — enable WAL archiving
archive_mode = on
archive_command = 'pgbackrest --stanza=yourdb archive-push %p'
wal_level = replica
```

```bash
pgbackrest --stanza=yourdb stanza-create
pgbackrest --stanza=yourdb --type=full backup     # weekly full
pgbackrest --stanza=yourdb --type=diff backup      # daily differential (cron/timer)

# Restore to a point in time (e.g. 1 minute before a bad migration):
pgbackrest --stanza=yourdb --type=time \
  --target="2026-05-30 14:32:00" --delta restore
```

### 2. Nightly logical dump — portability + selective restore + archive

PITR restores the *whole cluster* to a *moment*. A logical `pg_dump` is the complement: portable (move to another PG version/host), the basis for **selective** restore (one tenant, see below), and good for long-term archival. Runs as the dedicated backup role (see `deployment.md`), encrypted, shipped off-site.

```bash
# Nightly (systemd timer / cron). Custom format = parallelizable, selective pg_restore.
pg_dump --format=custom --no-owner --dbname="$BACKUP_DSN" \
  | age -r "$BACKUP_AGE_PUBKEY" \            # encrypt in-transit-to-disk (age/gpg)
  > "/tmp/yourdb-$(date +%F).dump.age"
aws s3 cp "/tmp/yourdb-$(date +%F).dump.age" "s3://yourapp-pgbackup/logical/" \
  --endpoint-url "$S3_ENDPOINT"
shred -u "/tmp/yourdb-"*.dump.age           # don't leave plaintext-adjacent files on the box
```

> **Why both.** PITR alone can't easily extract one tenant or survive a PG-version jump; logical dumps alone have a 24h RPO and slow whole-cluster restore. Together: PITR for disaster recovery (low RPO), logical dumps for portability and selective/archival restore.

---

## Per-Tenant Export & Restore

The single-DB RLS design (`database-schema.md`) has one sharp edge for recovery:

> **Whole-DB PITR cannot restore ONE tenant in place.** If tenant A bulk-deletes their data, you can't rewind only tenant A — a PITR restore rewinds *everyone*, clobbering every other tenant's writes since that point. RLS gives you isolation at runtime, not isolation of recovery.

The answer is a **logical per-tenant export**, which doubles as the GDPR/offboarding data-export path. Because it runs through `store.InTenant`, RLS auto-scopes every query to the one tenant — it is structurally impossible to leak another tenant's rows into the export.

```go
// ExportTenant — dump one tenant's rows across all tenant-scoped tables.
// Runs via store.InTenant: RLS restricts every SELECT to current_tenant_id(),
// so the export cannot accidentally include another tenant's data.
func (s *Service) ExportTenant(ctx context.Context) (*TenantExport, error) {
    var out TenantExport
    err := s.store.InTenant(ctx, func(q *db.Queries) error {
        var e error
        if out.Users, e = q.ExportUsers(ctx); e != nil { return e }
        if out.Roles, e = q.ExportRoles(ctx); e != nil { return e }
        if out.Content, e = q.ExportContent(ctx); e != nil { return e }
        if out.Media, e = q.ExportMedia(ctx); e != nil { return e } // metadata; bytes in R2
        if out.Audit, e = q.ExportAudit(ctx); e != nil { return e }
        return nil
    })
    return &out, err
}
```

**Selective restore** (undo one tenant's mistake without touching others):
1. Restore the latest backup into a **scratch/staging** database (never production).
2. Export the affected tenant from staging (the function above, pointed at staging).
3. Re-import those rows into production for that `tenant_id` (within a transaction, RLS-scoped).

> Per-tenant export also satisfies **GDPR data portability** ("give me my data") and **offboarding** ("delete everything for tenant X" → export for the record, then cascade-delete via `ON DELETE CASCADE` on `tenant_id`).

---

## R2/S3 Objects

> **Originals are kept permanently with object versioning — they are a backup artifact.** (This corrects an earlier "expire originals after 30 days" note in `media-upload.md`/`deployment.md`: that traded away recoverability. Originals are the only authoritative copy of raw bytes; keeping them lets you re-encode at any time — new variant sizes, better AVIF quality, or recovery if a variant is lost.)

- **Originals bucket**: lifecycle = keep; enable **object versioning** so an overwrite or accidental delete is recoverable. This is the object-storage equivalent of PITR.
- **Variants bucket (AVIF)**: *not* separately backed up — every variant is regenerable by re-running the converter (`media-upload.md`) over the retained original. If a variant is lost or you change sizes, re-encode.
- Cross-region replication of the originals bucket is optional belt-and-suspenders for region failure.

---

## Cross-Store Consistency on Restore

PG and R2 are backed up independently, so a restore can leave them at different moments. Restore in this order and the mismatch is benign:

1. **Restore PostgreSQL** (PITR to target time, or logical restore).
2. **Verify R2 objects** the restored `media` rows reference still exist. They almost always do, because originals are kept permanently + versioned (above) — a `media` row from any point in time still points at a retained original.
3. **Re-encode missing variants** if needed (derived, cheap to rebuild from the original).
4. **Rebuild KV** tenant config from the restored PG (it's derived); sessions simply expire and users re-login.

> The danger case — a `media` row pointing at an object that no longer exists — is exactly what permanent original retention prevents. This is *why* the originals decision was reversed: backup consistency depends on it.

---

## Security — a backup is a cross-tenant breach surface

> One backup file contains **every tenant's data and all password hashes** in a single artifact. A leaked backup is more catastrophic than a leaked request, because it bypasses RLS entirely (RLS protects the live DB, not a dump). Treat backups as the highest-sensitivity asset:

- **Encrypt** at rest (pgBackRest `aes-256-cbc`; `age`/`gpg` for dumps) and in transit (TLS to the off-site repo).
- **Store off the VPS** — a different provider/account ideally, so a compromise or loss of the VPS (or its cloud account) doesn't take the backups with it.
- **Dedicated backup role**, never `app_user` (see `deployment.md`): replication/read for PITR, read-all for logical dumps. The app's runtime credentials must not be able to read or delete backups.
- **Access-control + audit** the backup bucket; restrict who can decrypt. Restore access is effectively god-mode over all tenants.

---

## RPO / RTO + Restore Drills

Document the targets and **test them on a schedule** — a backup that has never been restored is a hypothesis, not a backup.

| Target | Value (tune per project) | Mechanism |
|---|---|---|
| **RPO** (max data loss) | ~5 min | WAL archiving (PITR) |
| **RTO** (max downtime) | ~1 h | base backup + WAL replay onto standby/new host |
| Logical archive | 24 h | nightly encrypted dump |

> **Scheduled restore drill (quarterly, automatable):** restore the latest backup into a throwaway instance, run the RLS isolation assertions (`database-schema.md` → "Tenant Isolation Testing") against it, spot-check row counts and a per-tenant export, then tear it down. This catches silent backup corruption, a broken `archive_command`, and missing-object drift *before* you need them in an incident.
