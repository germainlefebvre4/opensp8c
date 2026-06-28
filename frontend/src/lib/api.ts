import axios from 'axios'

const baseURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export const api = axios.create({ baseURL })

api.interceptors.response.use(
  r => r,
  err => {
    if (axios.isAxiosError(err) && err.response) {
      const data = err.response.data
      let message: string
      if (typeof data === 'string' && data.trim()) {
        message = data.trim()
      } else if (data && typeof data === 'object') {
        message = (data as Record<string, unknown>).error as string
          ?? (data as Record<string, unknown>).message as string
          ?? err.message
      } else {
        message = err.message
      }
      return Promise.reject(new Error(message))
    }
    return Promise.reject(err)
  }
)

export const wsURL = (path: string) => {
  const base = baseURL.replace(/^http/, 'ws')
  return `${base}${path}`
}
