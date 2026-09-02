import type { PatchSeasonRuleRequest, SeasonRule } from "@casa-dana/api"
import { Trash2 } from "lucide-react"
import { useCallback, useEffect, useRef, useState } from "react"

import { Input } from "@/components/ui/input"

import { centsToEuroInput, euroInputToCents } from "./money"

// Retouching a rule usually means changing several fields in a row: rename it,
// move both dates, set the price. Sending each one the moment it commits turned
// a single edit into four PATCHes — and four lines in the activity log. Field
// commits are accumulated instead and sent as one patch once the row has been
// quiet for this long.
const SAVE_DEBOUNCE_MS = 700

interface RuleRowProps {
  rule: SeasonRule
  onSave: (patch: PatchSeasonRuleRequest) => void
  onDelete: () => void
}

export function RuleRow({ rule, onSave, onDelete }: RuleRowProps) {
  const [label, setLabel] = useState(rule.label)
  const [startDate, setStartDate] = useState(rule.start_date)
  const [endDate, setEndDate] = useState(rule.end_date)
  const [price, setPrice] = useState(() => centsToEuroInput(rule.price_cents))

  const pending = useRef<PatchSeasonRuleRequest>({})
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const cancelTimer = () => {
    if (timer.current !== null) {
      clearTimeout(timer.current)
      timer.current = null
    }
  }

  const flush = useCallback(() => {
    cancelTimer()
    const patch = pending.current
    pending.current = {}
    if (Object.keys(patch).length > 0) onSave(patch)
  }, [onSave])

  // The card passes a fresh closure every render, so `flush` is never stable —
  // the unmount effect reads the latest one through a ref rather than
  // re-subscribing (and firing early) on each render.
  const flushRef = useRef(flush)
  flushRef.current = flush

  // Switching villa or navigating away unmounts the row mid-debounce; the edit
  // the user already made should still land.
  useEffect(() => () => flushRef.current(), [])

  const queue = (patch: PatchSeasonRuleRequest) => {
    pending.current = { ...pending.current, ...patch }
    cancelTimer()
    timer.current = setTimeout(() => flushRef.current(), SAVE_DEBOUNCE_MS)
  }

  // Once a save lands the refetched rule is the truth, so drop whatever the
  // inputs were holding — unless an edit is still queued, in which case a
  // background refetch would otherwise wipe what the user is typing.
  useEffect(() => {
    if (Object.keys(pending.current).length > 0) return
    setLabel(rule.label)
    setStartDate(rule.start_date)
    setEndDate(rule.end_date)
    setPrice(centsToEuroInput(rule.price_cents))
  }, [rule])

  const commitLabel = () => {
    const next = label.trim()
    // The schema requires 1..120 chars — an emptied field reverts rather than
    // sending a request the API would reject.
    if (!next) {
      setLabel(rule.label)
      return
    }
    if (next !== rule.label) queue({ label: next })
  }

  const commitPrice = () => {
    const cents = euroInputToCents(price)
    setPrice(centsToEuroInput(cents))
    if (cents !== rule.price_cents) queue({ price_cents: cents })
  }

  const commitStartDate = (value: string) => {
    setStartDate(value)
    if (value && value !== rule.start_date) queue({ start_date: value })
  }

  const commitEndDate = (value: string) => {
    setEndDate(value)
    if (value && value !== rule.end_date) queue({ end_date: value })
  }

  // Dropping the queued patch first: PATCHing a row we are about to delete
  // only earns a 404 and a misleading error toast.
  const handleDelete = () => {
    cancelTimer()
    pending.current = {}
    onDelete()
  }

  return (
    <div className="grid grid-cols-1 items-center gap-3 sm:grid-cols-[1.4fr_1fr_1fr_0.8fr_auto]">
      <Input
        aria-label="Nom de la règle"
        value={label}
        onChange={(e) => setLabel(e.target.value)}
        onBlur={commitLabel}
      />
      <Input
        type="date"
        aria-label="Date de début"
        value={startDate}
        onChange={(e) => commitStartDate(e.target.value)}
      />
      <Input
        type="date"
        aria-label="Date de fin"
        value={endDate}
        onChange={(e) => commitEndDate(e.target.value)}
      />
      <Input
        type="number"
        min={0}
        aria-label="Prix par nuit en euros"
        value={price}
        onChange={(e) => setPrice(e.target.value)}
        onBlur={commitPrice}
      />
      <button
        type="button"
        onClick={handleDelete}
        aria-label="Supprimer la règle"
        className="text-on-surface-variant hover:bg-error-container hover:text-on-error-container justify-self-start rounded-md p-1.5"
      >
        <Trash2 className="size-3.5" />
      </button>
    </div>
  )
}
