import { useGetVillaReviewMeta } from "@casa-dana/api"

import { AdminCard } from "@/components/admin/ui/admin-card"
import { EmptyState } from "@/components/admin/ui/empty-state"

const BREAKDOWN_FIELDS = [
  { key: "cleanliness", label: "Propreté" },
  { key: "comfort", label: "Confort" },
  { key: "location", label: "Emplacement" },
  { key: "host", label: "Hôte" },
  { key: "value", label: "Rapport qualité-prix" },
] as const

interface RatingBreakdownCardProps {
  property: "casadana" | "casacasay"
}

export default function RatingBreakdownCard({ property }: RatingBreakdownCardProps) {
  const { data: meta, isPending } = useGetVillaReviewMeta(property)

  return (
    <AdminCard
      title="Détail des notes"
      sub="Calculé à partir des avis publiés — publier ou masquer un avis fait bouger ces chiffres."
      flush={!meta}
    >
      {!meta ? (
        <EmptyState message={isPending ? "Chargement…" : "Notes indisponibles."} />
      ) : (
        <div className="flex flex-col gap-6 sm:flex-row sm:items-start sm:gap-10">
          <div className="sm:w-44 sm:shrink-0">
            {/* display_avg reads 0 with nothing approved, so the count is what tells
                an empty villa apart from a genuine average. */}
            {meta.display_count === 0 ? (
              <p className="text-on-surface-variant text-[13.5px]">
                Aucun avis publié pour le moment.
              </p>
            ) : (
              <>
                <p className="text-on-surface font-mono text-3xl font-bold">
                  {meta.display_avg.toFixed(2)}
                  <span className="text-on-surface-variant ml-1 text-base font-normal">/5</span>
                </p>
                <p className="text-on-surface-variant mt-1.5 text-[12.5px]">
                  Sur {meta.display_count} avis publié{meta.display_count === 1 ? "" : "s"}
                </p>
              </>
            )}
          </div>

          <div className="grid flex-1 gap-3">
            {BREAKDOWN_FIELDS.map(({ key, label }) => (
              <CategoryBar key={key} label={label} score={meta.breakdown[key]} />
            ))}
          </div>
        </div>
      )}
    </AdminCard>
  )
}

function CategoryBar({ label, score }: { label: string; score: number | null }) {
  return (
    <div className="grid grid-cols-[100px_1fr_auto] items-center gap-3 text-[12.5px]">
      <span className="text-on-surface-variant font-semibold">{label}</span>
      {score === null ? (
        <span className="text-on-surface-variant/70 col-span-2 italic">Pas encore noté</span>
      ) : (
        <>
          <div className="bg-outline-variant h-1.5 overflow-hidden rounded-full" aria-hidden>
            <span
              className="bg-secondary block h-full rounded-full"
              style={{ width: `${(score / 5) * 100}%` }}
            />
          </div>
          <span className="text-on-surface w-9 text-right font-mono">{score.toFixed(2)}</span>
        </>
      )}
    </div>
  )
}
