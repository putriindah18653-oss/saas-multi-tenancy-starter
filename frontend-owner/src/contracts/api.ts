/**
 * API Contract Types
 *
 * Single source of truth for all request/response types shared between the
 * backend API and frontend-owner dashboard. Every exported type documents the
 * shape the backend sends and the frontend expects.
 *
 * ## Envelope Pattern
 * Every API response is wrapped in a SuccessEnvelope<T> (success path) or
 * ErrorEnvelope (failure path). The `success` discriminator determines which
 * shape to expect.
 *
 * ## Error Codes
 * Map to HTTP status codes and drive UI error messages. See individual
 * documentation for trigger conditions.
 */

// ── Error Types ────────────────────────────────────────────────────────────

/**
 * Machine-readable error codes returned by the backend.
 *
 * | Code                     | HTTP | Trigger |
 * |--------------------------|------|---------|
 * | `unauthorized`           | 401  | Missing or expired access token |
 * | `forbidden`              | 403  | Valid token but insufficient permissions |
 * | `tenant_required`        | 400  | Request requires a tenant context but none was provided |
 * | `tenant_access_denied`   | 403  | User does not belong to requested tenant |
 * | `tenant_check_failed`    | 500  | Infrastructure failure checking tenant membership |
 * | `password_change_required`| 403 | User must change password before accessing dashboard |
 * | `validation_failed`      | 422  | Request payload failed validation |
 * | `invalid_upload`         | 400  | Uploaded file rejected (size, type, corruption) |
 * | `image_required`         | 400  | Upload endpoint called without an image file |
 * | `image_convert_failed`   | 500  | Server-side image processing failed |
 * | `upload_access_denied`   | 403  | User lacks permission to upload to this path |
 * | `rate_limited`           | 429  | Too many requests; retry after Retry-After seconds |
 * | `service_unavailable`    | 503  | Backend down or overloaded |
 * | `not_found`              | 404  | Requested resource does not exist |
 */
export type ErrorCode =
  | 'unauthorized'
  | 'forbidden'
  | 'tenant_required'
  | 'tenant_access_denied'
  | 'tenant_check_failed'
  | 'password_change_required'
  | 'validation_failed'
  | 'invalid_upload'
  | 'image_required'
  | 'image_convert_failed'
  | 'upload_access_denied'
  | 'rate_limited'
  | 'service_unavailable'
  | 'not_found'

/**
 * An API error payload. `code` is a known ErrorCode.
 * Use `(string & {})` instead of `| string` to preserve literal narrowing
 * in consumers that switch on known codes while accepting unknown future codes.
 */
export type ApiError = {
  code: ErrorCode | (string & {})
  message: string
}

/**
 * Success response wrapper. Every successful API response has `success: true`
 * and a `data` payload.
 */
export type SuccessEnvelope<T> = {
  readonly success: true
  readonly data: T
  readonly request_id?: string
}

/**
 * Error response wrapper. `success` is always `false`.
 * The `error.code` drives UI error handling (redirects, toast messages, etc.).
 */
export type ErrorEnvelope = {
  readonly success: false
  readonly error: ApiError
  readonly request_id?: string
}

/**
 * Backward-compatible alias for SuccessEnvelope.
 * @deprecated Use `SuccessEnvelope<T>` for clarity.
 */
export type Envelope<T> = SuccessEnvelope<T>

// ── Core Entities ──────────────────────────────────────────────────────────

/**
 * Authenticated user profile from `/me` and `/auth/refresh`.
 *
 * `display_name` is the canonical human-readable name (resolved by backend).
 * `full_name` and `name` are legacy fields kept for backward compatibility
 * with older backend versions. Prefer `display_name` when available.
 */
export type UserProfile = {
  readonly id: string
  readonly email: string
  /** Legacy — prefer `display_name`. */
  readonly name?: string
  /** Legacy — prefer `display_name`. */
  readonly full_name?: string
  readonly phone?: string
  readonly address?: string
  readonly avatar_url?: string
  readonly bio?: string
  readonly app_role?: string
  /** If true, user must change password before accessing dashboard. */
  readonly must_change_password?: boolean
  readonly is_active?: boolean
  /** Fine-grained permissions from `/me/permissions`. Falls back to role-based if absent. */
  readonly permissions?: string[]
  /** Tenant memberships for this user (optional — populated by `/me`). */
  readonly tenant_memberships?: TenantMembership[]
}

/**
 * A single tenant membership record from `/me`.
 */
export type TenantMembership = {
  readonly id?: string
  readonly tenant_id: string
  readonly tenant_name?: string
  readonly tenant_slug?: string
  readonly role: string
  readonly is_active?: boolean
}

/**
 * A tenant (customer organization).
 *
 * `status` is restricted to known values. Use `(string & {})` to keep literal
 * narrowing while accepting unknown future states from the backend.
 */
export type Tenant = {
  readonly id: string
  readonly name: string
  readonly slug: string
  readonly status: 'active' | 'suspended' | 'deleted' | (string & {})
}

/**
 * A user who belongs to a tenant (for tenant-user management pages).
 */
export type TenantMember = {
  readonly id: string
  readonly user_id: string
  readonly name: string
  readonly email: string
  readonly tenant_id: string
  readonly role: string
  readonly is_active: boolean
}

/**
 * Metadata for an uploaded file (avatar, logo).
 */
export type UploadResult = {
  readonly filename: string
  readonly url_path: string
  readonly size: number
}

/**
 * A single audit log entry.
 */
export type AuditEntry = {
  readonly id: string
  readonly actor_user_id?: string
  readonly tenant_id?: string
  readonly action: string
  readonly resource_type: string
  readonly resource_id?: string
  readonly metadata: Readonly<Record<string, unknown>>
  readonly ip_address: string
  readonly user_agent: string
  readonly created_at: string
}

/**
 * Company-level platform settings for the owner.
 */
export type AppCompanySettings = {
  readonly display_name: string
  readonly legal_name: string
  readonly logo_url: string
  readonly website_url: string
  readonly support_email: string
  readonly support_phone: string
  readonly timezone: string
  readonly locale: string
  readonly currency: string
  readonly address: string
  readonly metadata: Readonly<Record<string, unknown>>
}
