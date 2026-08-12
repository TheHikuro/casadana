import { createFileRoute, useNavigate } from "@tanstack/react-router"

import BasePricingCard from "@/components/admin/pricing/base-pricing-card"
import SeasonRulesCard from "@/components/admin/pricing/season-rules-card"

const PROPERTIES = ["casadana", "casacasay"] as const
type Property = (typeof PROPERTIES)[number]

interface PricingSearch {
  property: Property
}

function isProperty(value: unknown): value is Property {
  return typeof value === "string" && (PROPERTIES as ReadonlyArray<string>).includes(value)
}

function validatePricingSearch(search: Record<string, unknown>): PricingSearch {
  return { property: isProperty(search.property) ? search.property : "casadana" }
}

export const Route = createFileRoute("/admin/_authed/pricing")({
  validateSearch: validatePricingSearch,
  component: PricingPage,
})

const PROPERTY_LABELS: Record<Property, string> = {
  casadana: "Casa DaNa",
  casacasay: "Casa CasAy",
}

function PricingPage() {
  const { property } = Route.useSearch()
  const navigate = useNavigate({ from: Route.fullPath })

  const switchProperty = (nextProperty: Property) => {
    navigate({ search: (prev) => ({ ...prev, property: nextProperty }) })
  }

  return (
    <div>
      <div className="mb-7 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h2 className="text-on-surface text-2xl font-bold">Pricing</h2>
          <p className="text-on-surface-variant mt-1 text-[13.5px]">
            Base rate, fees and seasonal overrides for {PROPERTY_LABELS[property]}.
          </p>
        </div>
        <div className="bg-surface-container flex gap-1 rounded-lg p-1">
          {PROPERTIES.map((p) => (
            <button
              key={p}
              type="button"
              onClick={() => switchProperty(p)}
              className={
                p === property
                  ? "bg-primary text-on-primary rounded-md px-3 py-1.5 text-[13px] font-medium"
                  : "text-on-surface-variant rounded-md px-3 py-1.5 text-[13px] font-medium"
              }
            >
              {PROPERTY_LABELS[p]}
            </button>
          ))}
        </div>
      </div>

      {/* Keyed on the property so both cards drop their local form state on a
          switch instead of showing the previous villa's numbers. */}
      <div className="flex flex-col gap-5">
        <BasePricingCard key={property} property={property} />
        <SeasonRulesCard key={property} property={property} />
      </div>
    </div>
  )
}
