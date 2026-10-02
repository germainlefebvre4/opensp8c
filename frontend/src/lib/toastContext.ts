import { createContext } from 'react'

export type ToastVariant = 'success' | 'error' | 'warning'

export interface ToastOptions {
  title: string
  variant?: ToastVariant
  /** Display time in ms (default 4000). */
  duration?: number
}

export interface ToastContextValue {
  toast: (options: ToastOptions) => void
}

export const ToastContext = createContext<ToastContextValue | null>(null)
