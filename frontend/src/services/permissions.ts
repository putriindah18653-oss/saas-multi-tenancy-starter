export function hasPermission(current: string[] | undefined | null, required: string): boolean {
  if (!current || current.length === 0) return false
  return current.includes(required)
}

export function hasAnyPermission(current: string[] | undefined | null, required: string[]): boolean {
  if (!current || current.length === 0) return false
  return required.some((perm) => current.includes(perm))
}

export function hasAllPermissions(current: string[] | undefined | null, required: string[]): boolean {
  if (!current || current.length === 0) return false
  return required.every((perm) => current.includes(perm))
}
