import {
  type PatchSeasonRuleRequest,
  getListVillaSeasonRulesQueryKey,
  useCreateVillaSeasonRule,
  useDeleteSeasonRule,
  useGetVillaPricingSettings,
  useListVillaSeasonRules,
  usePatchSeasonRule,
} from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
import { format } from "date-fns"
import { Plus } from "lucide-react"

import { AdminCard } from "@/components/admin/ui/admin-card"
import { EmptyState } from "@/components/admin/ui/empty-state"
import { Button } from "@/components/ui/button"
import { useToast } from "@/components/ui/toast"

import { type Property, defaultBasePriceCents } from "./defaults"
import { RuleRow } from "./rule-row"

interface SeasonRulesCardProps {
  property: Property
}

export default function SeasonRulesCard({ property }: SeasonRulesCardProps) {
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const { data, isPending } = useListVillaSeasonRules(property)
  // Deduped with the base-pricing card's own query; a new rule starts at the
  // villa's current base price.
  const { data: settings } = useGetVillaPricingSettings(property)

  // The slug is baked into the URL-shaped query key, so it has to be passed
  // here — a no-arg call would build a key matching nothing.
  const rulesQueryKey = getListVillaSeasonRulesQueryKey(property)

  const { mutate: createRule } = useCreateVillaSeasonRule({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: rulesQueryKey })
        toast("Règle ajoutée")
      },
      onError: () => toast("Impossible d'ajouter la règle"),
    },
  })

  const { mutate: patchRule } = usePatchSeasonRule({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: rulesQueryKey })
        toast("Règle mise à jour")
      },
      onError: () => toast("Impossible de mettre à jour la règle"),
    },
  })

  const { mutate: deleteRule } = useDeleteSeasonRule({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: rulesQueryKey })
        toast("Règle supprimée")
      },
      onError: () => toast("Impossible de supprimer la règle"),
    },
  })

  const handleAdd = () => {
    const today = format(new Date(), "yyyy-MM-dd")
    createRule({
      slug: property,
      data: {
        label: "Nouvelle règle",
        start_date: today,
        end_date: today,
        // Same default as the base-rate card, so a villa nobody has configured
        // yet still starts its first rule on a real price instead of €0.
        price_cents: settings?.base_price_cents || defaultBasePriceCents(property),
      },
    })
  }

  const handleSave = (id: string, patch: PatchSeasonRuleRequest) => {
    patchRule({ id, data: patch })
  }

  const rules = data?.rules ?? []

  return (
    <AdminCard
      title="Règles saisonnières"
      sub="Sur une date couverte par plusieurs règles, la plus chère l'emporte."
      flush
      action={
        <Button type="button" size="sm" variant="outline" onClick={handleAdd}>
          <Plus />
          Ajouter une règle
        </Button>
      }
    >
      {isPending ? (
        <EmptyState message="Chargement…" />
      ) : rules.length === 0 ? (
        <EmptyState message="Aucune règle saisonnière pour le moment." />
      ) : (
        <div className="flex flex-col gap-3 p-5">
          {rules.map((rule) => (
            <RuleRow
              key={rule.id}
              rule={rule}
              onSave={(patch) => handleSave(rule.id, patch)}
              onDelete={() => deleteRule({ id: rule.id })}
            />
          ))}
        </div>
      )}
    </AdminCard>
  )
}
