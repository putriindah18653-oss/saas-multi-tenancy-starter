import type { AxiosError } from 'axios'

type DevLogContext = {
  status?: number
  code?: string
}

export function toLogContext(error: unknown): DevLogContext {
  const axiosError = error as AxiosError<{ error?: { code?: string } }>
  return {
    status: axiosError.response?.status,
    code: axiosError.response?.data?.error?.code,
  }
}

export function devWarn(event: string, context: DevLogContext = {}) {
  if (!import.meta.env.DEV) return
  console.warn(event, context)
}