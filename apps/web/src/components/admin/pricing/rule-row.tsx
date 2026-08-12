import type { PatchSeasonRuleRequest, SeasonRule } from "@casa-dana/api"
import { Trash2 } from "lucide-react"
import { useEffect, useState } from "react"

import { Input } from "@/components/ui/input"

import { centsToEuroInput, euroInputToCents } from "./money"

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

  // Once a save lands the refetched rule is the truth, so drop whatever the
  // inputs were holding.
  useEffect(() => {
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
    if (next !== rule.label) onSave({ label: next })
  }

  const commitPrice = () => {
    const cents = euroInputToCents(price)
    setPrice(centsToEuroInput(cents))
    if (cents !== rule.price_cents) onSave({ price_cents: cents })
  }

  const commitStartDate = (value: string) => {
    setStartDate(value)
    if (value && value !== rule.start_date) onSave({ start_date: value })
  }

  const commitEndDate = (value: string) => {
    setEndDate(value)
    if (value && value !== rule.end_date) onSave({ end_date: value })
  }

  return (
    <div className="grid grid-cols-1 items-center gap-3 sm:grid-cols-[1.4fr_1fr_1fr_0.8fr_auto]">
      <Input
        aria-label="Rule label"
        value={label}
        onChange={(e) => setLabel(e.target.value)}
        onBlur={commitLabel}
      />
      <Input
        type="date"
        aria-label="Start date"
        value={startDate}
        onChange={(e) => commitStartDate(e.target.value)}
      />
      <Input
        type="date"
        aria-label="End date"
        value={endDate}
        onChange={(e) => commitEndDate(e.target.value)}
      />
      <Input
        type="number"
        min={0}
        aria-label="Nightly price in euros"
        value={price}
        onChange={(e) => setPrice(e.target.value)}
        onBlur={commitPrice}
      />
      <button
        type="button"
        onClick={onDelete}
        aria-label="Delete rule"
        className="text-on-surface-variant hover:bg-error-container hover:text-on-error-container justify-self-start rounded-md p-1.5"
      >
        <Trash2 className="size-3.5" />
      </button>
    </div>
  )
}
