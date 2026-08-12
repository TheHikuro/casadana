import { cn } from "@/lib/utils"

interface ToggleProps {
  checked: boolean
  onChange: (next: boolean) => void
  /** Accessible name — the toggle renders no visible label. */
  label: string
  disabled?: boolean
}

export function Toggle({ checked, onChange, label, disabled }: ToggleProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative h-[22px] w-[38px] shrink-0 rounded-full transition-colors disabled:opacity-50",
        checked ? "bg-primary" : "bg-outline-variant",
      )}
    >
      <span
        className={cn(
          "absolute top-0.5 left-0.5 size-[18px] rounded-full bg-white transition-transform",
          checked && "translate-x-4",
        )}
      />
    </button>
  )
}
