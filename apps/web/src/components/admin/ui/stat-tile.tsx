interface StatTileProps {
  label: string
  value: string | number
}

export function StatTile({ label, value }: StatTileProps) {
  return (
    <div className="border-outline-variant bg-surface rounded-lg border px-4 py-3.5">
      <p className="text-on-surface-variant text-[11px] font-semibold tracking-[0.06em] uppercase">
        {label}
      </p>
      <p className="text-on-surface mt-1.5 font-mono text-2xl font-bold">{value}</p>
    </div>
  )
}

/** Responsive row of stat tiles, matching the dashboard's `stat-row` grid. */
export function StatRow({ children }: { children: React.ReactNode }) {
  return (
    <div className="mb-5 grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-4">{children}</div>
  )
}
