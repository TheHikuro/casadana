import {
  type PricingSettings,
  getGetVillaPricingSettingsQueryKey,
  useGetVillaPricingSettings,
  usePutVillaPricingSettings,
} from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"

import { AdminCard } from "@/components/admin/ui/admin-card"
import { AdminField, FieldActions, FieldGrid } from "@/components/admin/ui/admin-field"
import { EmptyState } from "@/components/admin/ui/empty-state"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useToast } from "@/components/ui/toast"

import { DEFAULT_MIN_NIGHTS, type Property, defaultBasePriceCents } from "./defaults"
import { centsToEuroInput, euroInputToCents } from "./money"

interface BasePricingForm {
  basePrice: string
  minNights: string
  cleaningFee: string
  conciergeFee: string
}

const EMPTY_FORM: BasePricingForm = {
  basePrice: "",
  minNights: "",
  cleaningFee: "",
  conciergeFee: "",
}

// An unset base rate or minimum stay opens on its default rather than on the
// zero the API reports for a villa nobody has configured yet. The two fees are
// left at zero on purpose: a villa that charges neither is a real answer, and
// the public panel already hides a zero fee.
function toForm(settings: PricingSettings, property: Property): BasePricingForm {
  return {
    basePrice: centsToEuroInput(settings.base_price_cents || defaultBasePriceCents(property)),
    minNights: String(settings.min_nights || DEFAULT_MIN_NIGHTS),
    cleaningFee: centsToEuroInput(settings.cleaning_fee_cents),
    conciergeFee: centsToEuroInput(settings.concierge_fee_cents),
  }
}

interface BasePricingCardProps {
  property: Property
}

export default function BasePricingCard({ property }: BasePricingCardProps) {
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const { data: settings, isPending } = useGetVillaPricingSettings(property)
  const [form, setForm] = useState<BasePricingForm>(EMPTY_FORM)

  // Re-seed from the server copy whenever it changes — the first load, and the
  // refetch that follows a save. Switching villa remounts the card entirely.
  useEffect(() => {
    if (settings) setForm(toForm(settings, property))
  }, [settings, property])

  const { mutate: saveSettings, isPending: isSaving } = usePutVillaPricingSettings({
    mutation: {
      onSuccess: () => {
        // The slug is baked into the URL-shaped query key, so it has to be
        // passed here — a no-arg call would build a key matching nothing.
        queryClient.invalidateQueries({ queryKey: getGetVillaPricingSettingsQueryKey(property) })
        toast("Pricing saved")
      },
      onError: () => toast("Could not save pricing"),
    },
  })

  const handleSave = () => {
    const minNights = Number.parseInt(form.minNights, 10)
    saveSettings({
      slug: property,
      data: {
        base_price_cents: euroInputToCents(form.basePrice),
        // The field opens pre-filled, but it can still be cleared by hand and
        // the PUT schema rejects anything below 1 — floor it here.
        min_nights: Number.isFinite(minNights)
          ? Math.max(DEFAULT_MIN_NIGHTS, minNights)
          : DEFAULT_MIN_NIGHTS,
        cleaning_fee_cents: euroInputToCents(form.cleaningFee),
        concierge_fee_cents: euroInputToCents(form.conciergeFee),
      },
    })
  }

  return (
    <AdminCard
      title="Base rate & fees"
      sub="Applied whenever no seasonal rule matches."
      flush={isPending}
    >
      {isPending ? (
        <EmptyState message="Loading…" />
      ) : (
        <>
          <FieldGrid>
            <AdminField label="Base nightly price (€)" htmlFor="basePrice">
              <Input
                id="basePrice"
                type="number"
                min={0}
                value={form.basePrice}
                onChange={(e) => setForm((prev) => ({ ...prev, basePrice: e.target.value }))}
              />
            </AdminField>
            <AdminField label="Minimum nights" htmlFor="minNights">
              <Input
                id="minNights"
                type="number"
                min={1}
                value={form.minNights}
                onChange={(e) => setForm((prev) => ({ ...prev, minNights: e.target.value }))}
              />
            </AdminField>
            <AdminField label="Cleaning fee (€)" htmlFor="cleaningFee">
              <Input
                id="cleaningFee"
                type="number"
                min={0}
                value={form.cleaningFee}
                onChange={(e) => setForm((prev) => ({ ...prev, cleaningFee: e.target.value }))}
              />
            </AdminField>
            <AdminField label="Concierge fee (€)" htmlFor="conciergeFee">
              <Input
                id="conciergeFee"
                type="number"
                min={0}
                value={form.conciergeFee}
                onChange={(e) => setForm((prev) => ({ ...prev, conciergeFee: e.target.value }))}
              />
            </AdminField>
          </FieldGrid>
          <FieldActions>
            <Button type="button" disabled={isSaving} onClick={handleSave}>
              {isSaving ? "Saving…" : "Save base pricing"}
            </Button>
          </FieldActions>
        </>
      )}
    </AdminCard>
  )
}
