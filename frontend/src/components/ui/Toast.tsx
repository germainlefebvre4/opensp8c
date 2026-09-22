import { useCallback, useState, type ReactNode } from 'react'
import * as ToastPrimitive from '@radix-ui/react-toast'
import { CheckCircle2, AlertCircle } from 'lucide-react'
import { ToastContext, type ToastOptions } from '../../lib/toastContext'

interface ToastEntry extends ToastOptions {
  id: number
}

let nextId = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastEntry[]>([])

  const toast = useCallback((options: ToastOptions) => {
    const id = nextId++
    setToasts(prev => [...prev, { id, ...options }])
  }, [])

  const dismiss = useCallback((id: number) => {
    setToasts(prev => prev.filter(t => t.id !== id))
  }, [])

  return (
    <ToastContext.Provider value={{ toast }}>
      <ToastPrimitive.Provider swipeDirection="right">
        {children}
        {toasts.map(t => (
          <ToastPrimitive.Root
            key={t.id}
            duration={4000}
            onOpenChange={open => { if (!open) dismiss(t.id) }}
            className="bg-white rounded-lg shadow-xl border border-slate-200 px-4 py-3 flex items-center gap-2"
          >
            {t.variant === 'error' ? (
              <AlertCircle size={16} className="text-red-500 shrink-0" />
            ) : (
              <CheckCircle2 size={16} className="text-emerald-500 shrink-0" />
            )}
            <ToastPrimitive.Title className="text-xs font-medium text-slate-800">
              {t.title}
            </ToastPrimitive.Title>
          </ToastPrimitive.Root>
        ))}
        <ToastPrimitive.Viewport className="fixed bottom-4 right-4 z-[100] flex flex-col gap-2 w-[320px] max-w-[calc(100vw-2rem)] outline-none" />
      </ToastPrimitive.Provider>
    </ToastContext.Provider>
  )
}
