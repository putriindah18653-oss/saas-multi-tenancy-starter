/**
 * API Contract Types — Tenant Dashboard
 *
 * Single source of truth for all request/response types shared between the
 * backend API and frontend-tenant dashboard. Every exported type documents the
 * shape the backend sends and the frontend expects.
 *
 * ## Envelope Pattern
 * Every API response is wrapped in either `SuccessEnvelope<T>` (success) or
 * `ErrorEnvelope` (failure). The `success` discriminator determines the shape.
 *
 * ## Error Codes
 * | Code                     | HTTP | Trigger |
 * |--------------------------|------|---------|
 * | `unauthorized`           | 401  | Missing/expired access token |
 * | `forbidden`              | 403  | Valid token but insufficient permissions |
 * | `tenant_required`        | 400  | Request requires tenant context |
 * | `tenant_access_denied`   | 403  | User does not belong to requested tenant |
 * | `tenant_check_failed`    | 500  | Infrastructure failure checking membership |
 * | `password_change_required`| 403 | User must change password; triggers redirect |
 * | `validation_failed`      | 422  | Request payload failed validation |
 * | `invalid_upload`         | 400  | Uploaded file rejected (size, type, corruption) |
 * | `image_required`         | 400  | Upload endpoint called without image file |
 * | `image_convert_failed`   | 500  | Server-side image processing failed |
 * | `upload_access_denied`   | 403  | User lacks upload permission |
 * | `rate_limited`           | 429  | Too many requests; retry after Retry-After |
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

/** Error response body. code uses `(string & {})` to preserve literal narrowing. */
export type ApiError = {
  code: ErrorCode | (string & {})
  message: string
}

/** Success response wrapper. */
export type SuccessEnvelope<T> = {
  readonly success: true
  readonly data: T
  readonly request_id?: string
}

/** Error response wrapper. `success` is always false. */
export type ErrorEnvelope = {
  readonly success: false
  readonly error: ApiError
  readonly request_id?: string
}

/** @deprecated Use `SuccessEnvelope<T>` for clarity. */
export type Envelope<T> = SuccessEnvelope<T>

/**
 * Authenticated user profile. `display_name` is canonical; `name`/`full_name`
 * are legacy fields for backward compatibility with older backend versions.
 */
export type UserProfile = {
  readonly id: string
  readonly email: string
  readonly display_name?: string
  readonly name?: string
  readonly full_name?: string
  readonly phone?: string
  readonly address?: string
  readonly avatar_url?: string
  readonly bio?: string
  readonly app_role?: string
  readonly must_change_password?: boolean
  readonly is_active?: boolean
  readonly permissions?: string[]
  readonly tenant_memberships?: TenantMembership[]
}

/** A single tenant membership record from /me. */
export type TenantMembership = {
  readonly id?: string
  readonly tenant_id: string
  readonly tenant_name?: string
  readonly tenant_slug?: string
  readonly role: string
  readonly is_active?: boolean
}

/** A user within a tenant (for tenant-user management). */
export type TenantMember = {
  readonly id: string
  readonly user_id: string
  readonly name: string
  readonly email: string
  readonly tenant_id: string
  readonly role: string
  readonly is_active: boolean
}

/** Tenant-level settings (branding, regional, currency). */
export type TenantSettings = {
  readonly tenant_id: string
  readonly display_name: string
  readonly logo_url: string
  readonly timezone: string
  readonly locale: string
  readonly currency: string
  readonly metadata: Readonly<Record<string, unknown>>
}

/** Metadata for an uploaded file. */
export type UploadResult = {
  readonly filename: string
  readonly url_path: string
  readonly size: number
}

/** A single audit log entry. */
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

/** Shape used by catch blocks for Axios error responses. */
export type ApiErrorResponse = { response?: { data?: { error?: { message?: string } } } }
