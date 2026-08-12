import {
  type ReviewCategoryRatings,
  getGetVillaReviewMetaQueryKey,
  getListAdminReviewsQueryKey,
  getListVillaReviewsQueryKey,
  useCreateAdminReview,
} from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
import { Plus } from "lucide-react"
import { useState } from "react"
import { useForm } from "react-hook-form"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Field, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useToast } from "@/components/ui/toast"
import { cn } from "@/lib/utils"

const RATINGS = [1, 2, 3, 4, 5] as const

const CATEGORY_FIELDS = [
  { key: "cleanliness", label: "Cleanliness" },
  { key: "comfort", label: "Comfort" },
  { key: "location", label: "Location" },
  { key: "host", label: "Host" },
  { key: "value", label: "Value" },
] as const

type CategoryKey = (typeof CATEGORY_FIELDS)[number]["key"]

interface AddReviewFormValues {
  authorName: string
  meta: string
  rating: number
  body: string
  source: string
  // Held as strings so a blank input stays blank instead of collapsing to 0,
  // which the API rejects — scores have to be 1-5 or absent.
  categories: Record<CategoryKey, string>
}

const DEFAULT_VALUES: AddReviewFormValues = {
  authorName: "",
  meta: "",
  rating: 5,
  body: "",
  source: "",
  categories: { cleanliness: "", comfort: "", location: "", host: "", value: "" },
}

/** Undefined when nothing was scored, so the key is left off the payload entirely. */
function toCategoryRatings(values: Record<CategoryKey, string>): ReviewCategoryRatings | undefined {
  const scored: ReviewCategoryRatings = {}
  for (const { key } of CATEGORY_FIELDS) {
    const score = Number(values[key])
    if (values[key].trim() !== "" && Number.isFinite(score)) scored[key] = score
  }
  return Object.keys(scored).length > 0 ? scored : undefined
}

interface AddReviewDialogProps {
  property: "casadana" | "casacasay"
}

export default function AddReviewDialog({ property }: AddReviewDialogProps) {
  const [open, setOpen] = useState(false)
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const { register, handleSubmit, reset, watch, setValue } = useForm<AddReviewFormValues>({
    defaultValues: DEFAULT_VALUES,
  })
  const rating = watch("rating")

  const { mutate: createReview, isPending } = useCreateAdminReview({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: getListAdminReviewsQueryKey() })
        queryClient.invalidateQueries({ queryKey: getListVillaReviewsQueryKey(property) })
        // An approved review shifts the computed figures, so the breakdown card
        // has to refetch alongside the lists.
        queryClient.invalidateQueries({ queryKey: getGetVillaReviewMetaQueryKey(property) })
        toast("Review added")
        setOpen(false)
        reset(DEFAULT_VALUES)
      },
      onError: () => toast("Could not add review"),
    },
  })

  const onSubmit = (values: AddReviewFormValues) => {
    const authorName = values.authorName.trim()
    const body = values.body.trim()
    if (!authorName || !body) {
      toast("Fill in name and quote")
      return
    }
    createReview({
      data: {
        villa_slug: property,
        author_name: authorName,
        rating: values.rating,
        body,
        meta: values.meta.trim(),
        source: values.source.trim() || "Direct booking",
        status: "approved",
        featured: false,
        categories: toCategoryRatings(values.categories),
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button type="button" />}>
        <Plus />
        Add review
      </DialogTrigger>
      <DialogContent>
        <DialogTitle>Add review</DialogTitle>
        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-3">
          <Field>
            <FieldLabel htmlFor="reviewAuthorName">Guest name</FieldLabel>
            <Input id="reviewAuthorName" {...register("authorName")} />
          </Field>
          <Field>
            <FieldLabel htmlFor="reviewMeta">Meta line</FieldLabel>
            <Input
              id="reviewMeta"
              placeholder="Paris, France · Stayed June 2026"
              {...register("meta")}
            />
          </Field>
          <Field>
            <FieldLabel>Rating</FieldLabel>
            <div className="flex items-center gap-0.5">
              {RATINGS.map((n) => (
                <button
                  key={n}
                  type="button"
                  aria-label={n === 1 ? "Rate 1 star" : `Rate ${n} stars`}
                  aria-pressed={n === rating}
                  onClick={() => setValue("rating", n)}
                  className={cn(
                    "rounded-md px-1 py-0.5 text-lg leading-none",
                    n <= rating ? "text-on-surface" : "text-on-surface-variant/50",
                  )}
                >
                  ★
                </button>
              ))}
            </div>
          </Field>
          <Field>
            <FieldLabel>Category ratings (optional)</FieldLabel>
            <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-3">
              {CATEGORY_FIELDS.map(({ key, label }) => (
                <div key={key} className="flex flex-col gap-1">
                  <label
                    htmlFor={`reviewCategory-${key}`}
                    className="text-on-surface-variant text-[11.5px] font-semibold"
                  >
                    {label}
                  </label>
                  <Input
                    id={`reviewCategory-${key}`}
                    type="number"
                    min={1}
                    max={5}
                    step={1}
                    placeholder="1-5"
                    {...register(`categories.${key}`)}
                  />
                </div>
              ))}
            </div>
          </Field>
          <Field>
            <FieldLabel htmlFor="reviewBody">Quote</FieldLabel>
            <Textarea id="reviewBody" rows={4} {...register("body")} />
          </Field>
          <Field>
            <FieldLabel htmlFor="reviewSource">Source</FieldLabel>
            <Input id="reviewSource" placeholder="via Airbnb · Couple" {...register("source")} />
          </Field>
          <div className="mt-2 flex justify-end gap-2.5">
            <DialogClose render={<Button type="button" variant="outline" />}>Cancel</DialogClose>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Saving…" : "Save review"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
