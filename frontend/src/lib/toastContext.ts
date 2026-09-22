import { createContext } from 'react'

export type ToastVariant = 'success' | 'error'

export interface ToastOptions {
  title: string
  variant?: ToastVariant
}

export interface ToastContextValue {
  toast: (options: ToastOptions) => void
}

export const ToastContext = createContext<ToastContextValue | null>(null)
