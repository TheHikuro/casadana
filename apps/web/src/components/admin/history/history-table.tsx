import type { HistoryEvent, HistoryEventType } from "@casa-dana/api"
import { CalendarDays, Circle, type LucideIcon, Star, Tag, UserCog } from "lucide-react"

const TYPE_ICONS: Record<HistoryEventType, LucideIcon> = {
  reservation: CalendarDays,
  pricing: Tag,
  review: Star,
  owner: UserCog,
  system: Circle,
}

// The API enum can grow ahead of this map; an unknown type still renders.
const FALLBACK_ICON = Circle

const DATE_TIME_FORMAT = new Intl.DateTimeFormat(undefined, {
  dateStyle: "medium",
  timeStyle: "short",
})

function formatWhen(timestamp: string): string {
  const parsed = new Date(timestamp)
  return Number.isNaN(parsed.getTime()) ? timestamp : DATE_TIME_FORMAT.format(parsed)
}

export default function HistoryTable({ events }: { events: Array<HistoryEvent> }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[720px] border-collapse text-[13px]">
        <thead>
          <tr className="border-outline-variant bg-surface-container-low text-on-surface-variant border-b text-left text-[10.5px] font-semibold tracking-[0.08em] uppercase">
            <th className="px-5 py-2.5">When</th>
            <th className="px-5 py-2.5">Type</th>
            <th className="px-5 py-2.5">Event</th>
          </tr>
        </thead>
        <tbody>
          {events.map((event) => {
            const Icon = TYPE_ICONS[event.type] ?? FALLBACK_ICON
            const actor = event.actor_email.trim()
            return (
              <tr key={event.id} className="border-outline-variant border-b last:border-0">
                <td className="text-on-surface-variant px-5 py-3 font-mono whitespace-nowrap">
                  {formatWhen(event.created_at)}
                </td>
                <td className="px-5 py-3">
                  <span className="bg-surface-container-high text-on-surface-variant inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11.5px] font-semibold capitalize">
                    <Icon className="size-3.5 shrink-0" />
                    {event.type}
                  </span>
                </td>
                <td className="text-on-surface px-5 py-3">
                  {event.message}
                  {actor && (
                    <span className="text-on-surface-variant ml-2 font-mono text-[11.5px]">
                      {actor}
                    </span>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
