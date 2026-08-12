import {
  type ReviewMeta,
  getGetVillaReviewMetaQueryKey,
  useGetVillaReviewMeta,
  usePutVillaReviewMeta,
} from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
import { type FormEvent, useState } from "react"

import { AdminCard } from "@/components/admin/ui/admin-card"
import { AdminField, FieldActions, FieldGrid } from "@/components/admin/ui/admin-field"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useToast } from "@/components/ui/toast"

const BREAKDOWN_FIELDS = [
  { key: "cleanliness", label: "Cleanliness" },
  { key: "comfort", label: "Comfort" },
  { key: "location", label: "Location" },
  { key: "host", label: "Host" },
  { key: "value", label: "Value" },
] as const

type BreakdownKey = (typeof BREAKDOWN_FIELDS)[number]["key"]

// Held as strings so a half-typed number (or a cleared input) survives until
// submit instead of collapsing to NaN.
type BreakdownForm = Record<BreakdownKey | "display_avg" | "display_count", string>

const EMPTY_FORM: BreakdownForm = {
  cleanliness: "",
  comfort: "",
  location: "",
  host: "",
  value: "",
  display_avg: "",
  display_count: "",
}

function toForm(meta: ReviewMeta | undefined): BreakdownForm {
  if (!meta) return EMPTY_FORM
  return {
    cleanliness: String(meta.breakdown.cleanliness),
    comfort: String(meta.breakdown.comfort),
    location: String(meta.breakdown.location),
    host: String(meta.breakdown.host),
    value: String(meta.breakdown.value),
    display_avg: String(meta.display_avg),
    display_count: String(meta.display_count),
  }
}

interface RatingBreakdownCardProps {
  property: "casadana" | "casacasay"
}

export default function RatingBreakdownCard({ property }: RatingBreakdownCardProps) {
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const { data: meta } = useGetVillaReviewMeta(property)

  // Seeded from the query and re-seeded whenever the fetched meta changes, so
  // the form never shows another villa's figures. React Query's structural
  // sharing keeps `meta` referentially stable across refetches that return the
  // same data, so a background refetch can't clobber an in-progress edit.
  const [seededMeta, setSeededMeta] = useState(meta)
  const [form, setForm] = useState(() => toForm(meta))
  if (meta !== seededMeta) {
    setSeededMeta(meta)
    setForm(toForm(meta))
  }

  const setField = (key: keyof BreakdownForm, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const { mutate: saveMeta, isPending } = usePutVillaReviewMeta({
    mutation: {
      onSuccess: () => {
        // The meta key embeds the slug, so this one is invalidated exactly.
        queryClient.invalidateQueries({ queryKey: getGetVillaReviewMetaQueryKey(property) })
        toast("Rating breakdown saved")
      },
      onError: () => toast("Could not save the rating breakdown"),
    },
  })

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    saveMeta({
      slug: property,
      data: {
        display_avg: Number(form.display_avg) || 0,
        display_count: Number(form.display_count) || 0,
        breakdown: {
          cleanliness: Number(form.cleanliness) || 0,
          comfort: Number(form.comfort) || 0,
          location: Number(form.location) || 0,
          host: Number(form.host) || 0,
          value: Number(form.value) || 0,
        },
      },
    })
  }

  return (
    <AdminCard
      title="Rating breakdown"
      sub="Shown as the progress bars on the public reviews section."
    >
      <form onSubmit={onSubmit}>
        <FieldGrid>
          {BREAKDOWN_FIELDS.map(({ key, label }) => (
            <AdminField key={key} label={label} htmlFor={`breakdown-${key}`}>
              <Input
                id={`breakdown-${key}`}
                type="number"
                step={0.1}
                min={0}
                max={5}
                value={form[key]}
                onChange={(e) => setField(key, e.target.value)}
              />
            </AdminField>
          ))}
        </FieldGrid>

        <FieldGrid className="mt-4">
          <AdminField label="Average (displayed)" htmlFor="breakdown-display-avg">
            <Input
              id="breakdown-display-avg"
              type="number"
              step={0.01}
              min={0}
              max={5}
              value={form.display_avg}
              onChange={(e) => setField("display_avg", e.target.value)}
            />
          </AdminField>
          <AdminField label="Review count (displayed)" htmlFor="breakdown-display-count">
            <Input
              id="breakdown-display-count"
              type="number"
              step={1}
              min={0}
              value={form.display_count}
              onChange={(e) => setField("display_count", e.target.value)}
            />
          </AdminField>
        </FieldGrid>

        <FieldActions>
          <Button type="submit" disabled={isPending}>
            {isPending ? "Saving…" : "Save breakdown"}
          </Button>
        </FieldActions>
      </form>
    </AdminCard>
  )
}
