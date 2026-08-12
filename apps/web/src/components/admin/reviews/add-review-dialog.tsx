import {
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

interface AddReviewFormValues {
  authorName: string
  meta: string
  rating: number
  body: string
  source: string
}

const DEFAULT_VALUES: AddReviewFormValues = {
  authorName: "",
  meta: "",
  rating: 5,
  body: "",
  source: "",
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
