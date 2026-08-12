import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** Two-column form grid used by every settings card. */
export function FieldGrid({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("grid grid-cols-1 gap-4 sm:grid-cols-2", className)}>{children}</div>
}

interface AdminFieldProps {
  label: string
  htmlFor?: string
  /** Span both columns of the enclosing FieldGrid. */
  span2?: boolean
  children: ReactNode
}

export function AdminField({ label, htmlFor, span2, children }: AdminFieldProps) {
  return (
    <div className={cn("flex flex-col gap-1.5", span2 && "sm:col-span-2")}>
      <label htmlFor={htmlFor} className="text-on-surface-variant text-xs font-semibold">
        {label}
      </label>
      {children}
    </div>
  )
}

/** Right-aligned action row closing a settings card. */
export function FieldActions({ children }: { children: ReactNode }) {
  return <div className="mt-4 flex justify-end gap-2.5">{children}</div>
}
