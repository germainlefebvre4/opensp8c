import axios from 'axios'

const baseURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export const api = axios.create({ baseURL })

export const wsURL = (path: string) => {
  const base = baseURL.replace(/^http/, 'ws')
  return `${base}${path}`
}
