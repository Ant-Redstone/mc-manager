import { AnimatePresence, motion } from 'framer-motion'
import { useEffect } from 'react'
import type { ReactNode } from 'react'
import clsx from 'clsx'

type ButtonVariant = 'primary' | 'danger' | 'ghost'

const buttonVariants: Record<ButtonVariant, string> = {
  primary: 'bg-brand text-white shadow-glow hover:bg-brand-hover',
  danger: 'border border-status-error/40 bg-status-error/10 text-status-error hover:bg-status-error/20',
  ghost: 'border border-border-subtle bg-white/5 text-text-secondary hover:bg-white/10',
}

export function Button({
  children,
  onClick,
  type = 'button',
  variant = 'ghost',
  disabled = false,
  className,
  title,
}: {
  children: ReactNode
  onClick?: () => void
  type?: 'button' | 'submit'
  variant?: ButtonVariant
  disabled?: boolean
  className?: string
  title?: string
}) {
  return (
    <motion.button
      type={type}
      onClick={onClick}
      disabled={disabled}
      title={title}
      whileHover={disabled ? undefined : { scale: 1.03 }}
      whileTap={disabled ? undefined : { scale: 0.96 }}
      transition={{ type: 'spring', stiffness: 400, damping: 17 }}
      className={clsx(
        'inline-flex min-h-touch items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium',
        'transition-colors disabled:cursor-not-allowed disabled:opacity-40 disabled:shadow-none',
        buttonVariants[variant],
        className,
      )}
    >
      {children}
    </motion.button>
  )
}

export function Card({
  children,
  className,
  delay = 0,
}: {
  children: ReactNode
  className?: string
  delay?: number
}) {
  return (
    <motion.section
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, ease: 'easeOut', delay }}
      className={clsx('glass-card p-5 sm:p-6', className)}
    >
      {children}
    </motion.section>
  )
}

type BadgeTone = 'success' | 'neutral' | 'error' | 'info'

export function Badge({ children, tone = 'neutral' }: { children: ReactNode; tone?: BadgeTone }) {
  const tones: Record<BadgeTone, string> = {
    success: 'bg-status-success/15 text-status-success',
    error: 'bg-status-error/15 text-status-error',
    info: 'bg-status-info/15 text-status-info',
    neutral: 'bg-white/10 text-text-secondary',
  }
  return (
    <span
      className={clsx(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold uppercase tracking-wide',
        tones[tone],
      )}
    >
      {children}
    </span>
  )
}

export function Spinner({ className }: { className?: string }) {
  return (
    <span
      className={clsx(
        'inline-block size-4 animate-spin rounded-full border-2 border-white/20 border-t-brand',
        className,
      )}
    />
  )
}

export function StatusDot({ on }: { on: boolean }) {
  return (
    <span className="relative flex size-2.5">
      {on && (
        <span className="absolute inline-flex size-full animate-ping rounded-full bg-status-success opacity-60" />
      )}
      <span
        className={clsx('relative inline-flex size-2.5 rounded-full', on ? 'bg-status-success' : 'bg-text-muted')}
      />
    </span>
  )
}

export function Logo() {
  return (
    <div className="grid size-10 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-brand-light to-brand shadow-glow">
      <svg
        width="22"
        height="22"
        viewBox="0 0 64 64"
        fill="none"
        stroke="#fff"
        strokeWidth="3.4"
        strokeLinejoin="round"
        strokeLinecap="round"
      >
        <path d="M32 13 L50 22.5 L50 41.5 L32 51 L14 41.5 L14 22.5 Z" />
        <path d="M14 22.5 L32 32 L50 22.5" />
        <path d="M32 32 L32 51" />
      </svg>
    </div>
  )
}

export function Toast({ message, onDone }: { message: string | null; onDone: () => void }) {
  useEffect(() => {
    if (!message) return
    const id = window.setTimeout(onDone, 4000)
    return () => clearTimeout(id)
  }, [message, onDone])

  return (
    <AnimatePresence>
      {message && (
        <motion.div
          initial={{ opacity: 0, y: 24 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 24 }}
          className="fixed bottom-5 right-5 z-50 max-w-sm rounded-xl border border-status-error/40 bg-neutral-bg3/90 px-4 py-3 text-sm shadow-glow-lg backdrop-blur-md"
        >
          {message}
        </motion.div>
      )}
    </AnimatePresence>
  )
}
