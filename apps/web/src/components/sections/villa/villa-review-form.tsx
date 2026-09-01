import { ApiError, type ReviewCategoryRatings, useSubmitVillaReview } from "@casa-dana/api"
import { ArrowRight } from "lucide-react"
import { useState } from "react"
import { useForm } from "react-hook-form"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useToast } from "@/components/ui/toast"
import { cn } from "@/lib/utils"
import { m } from "@/paraglide/messages"

interface VillaReviewFormProps {
  villaSlug: string
  /**
   * The five per-category labels from the villa's own copy, in the order the
   * API's category keys are declared. Reused here so the form and the
   * breakdown bars above it never word the same category differently.
   */
  barLabels: Array<string>
}

const STARS = [1, 2, 3, 4, 5] as const

const CATEGORY_KEYS = [
  "cleanliness",
  "comfort",
  "location",
  "host",
  "value",
] as const satisfies ReadonlyArray<keyof ReviewCategoryRatings>

type CategoryKey = (typeof CATEGORY_KEYS)[number]

interface ReviewFormValues {
  authorName: string
  rating: number
  body: string
  /** 0 means "not rated" — the API wants 1..5 or the key left off entirely. */
  categories: Record<CategoryKey, number>
}

const DEFAULT_VALUES: ReviewFormValues = {
  authorName: "",
  rating: 5,
  body: "",
  categories: { cleanliness: 0, comfort: 0, location: 0, host: 0, value: 0 },
}

/** Undefined when the visitor scored nothing, so the key never reaches the API. */
function toCategoryRatings(scores: Record<CategoryKey, number>): ReviewCategoryRatings | undefined {
  const scored: ReviewCategoryRatings = {}
  for (const key of CATEGORY_KEYS) {
    if (scores[key] > 0) scored[key] = scores[key]
  }
  return Object.keys(scored).length > 0 ? scored : undefined
}

const ERROR_MESSAGES: Record<string, () => string> = {
  UNKNOWN_VILLA: m.villa_reviews_form_error_unknown_villa,
  INVALID_PAYLOAD: m.villa_reviews_form_error_invalid,
  VALIDATION: m.villa_reviews_form_error_invalid,
  // Its own wording rather than the generic failure: nothing is wrong with what
  // the visitor wrote, so telling them to fix it would send them in circles.
  RATE_LIMITED: m.villa_reviews_form_error_rate_limited,
}

interface StarPickerProps {
  value: number
  /** Builds each star's label, so the overall row and a category row can word it differently. */
  describeStar: (count: number) => string
  size: "sm" | "lg"
  onChange: (next: number) => void
}

function StarPicker({ value, describeStar, size, onChange }: StarPickerProps) {
  const glyphSize = size === "lg" ? "text-[26px]" : "text-[15px]"
  return (
    <div className="flex items-center gap-0.5">
      {STARS.map((n) => (
        <button
          key={n}
          type="button"
          aria-label={describeStar(n)}
          aria-pressed={n === value}
          onClick={() => onChange(n)}
          className={cn(
            "leading-none transition-colors",
            glyphSize,
            n <= value ? "text-secondary" : "text-outline-variant hover:text-secondary/60",
          )}
        >
          ★
        </button>
      ))}
    </div>
  )
}

const fieldLabelClassName =
  "text-on-surface-variant block font-mono text-[10px] tracking-[0.22em] uppercase"

const inputClassName =
  "text-primary placeholder:text-on-surface-variant/50 mt-1 h-auto w-full rounded-none border-0 bg-transparent px-0 py-0 text-[15px] shadow-none focus-visible:border-0 focus-visible:ring-0 md:text-[15px]"

export default function VillaReviewForm({ villaSlug, barLabels }: VillaReviewFormProps) {
  const { toast } = useToast()
  const [submitted, setSubmitted] = useState(false)
  const [topLevelError, setTopLevelError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<ReviewFormValues>({ defaultValues: DEFAULT_VALUES })

  const rating = watch("rating")
  const categories = watch("categories")

  const { mutate: submitReview, isPending } = useSubmitVillaReview({
    mutation: {
      onSuccess: () => {
        // Nothing to invalidate: the review lands `pending`, so neither the
        // public listing nor the computed figures have moved.
        setTopLevelError(null)
        setSubmitted(true)
        toast(m.villa_reviews_form_success_toast())
      },
      onError: (err) => {
        const code = err instanceof ApiError ? err.code : "NETWORK"
        const message = ERROR_MESSAGES[code] ?? m.villa_reviews_form_error_generic
        setTopLevelError(message())
      },
    },
  })

  const handleRating = (next: number) => setValue("rating", next, { shouldDirty: true })

  const handleCategory = (key: CategoryKey, next: number) => {
    // Clicking the score already set clears it: a visitor who tapped a
    // category by accident needs a way back to "not rated".
    const value = categories[key] === next ? 0 : next
    setValue(`categories.${key}`, value, { shouldDirty: true })
  }

  const onSubmit = (values: ReviewFormValues) => {
    setTopLevelError(null)
    submitReview({
      slug: villaSlug,
      data: {
        author_name: values.authorName.trim(),
        rating: values.rating,
        body: values.body.trim(),
        categories: toCategoryRatings(values.categories),
      },
    })
  }

  const panelClassName = "bg-background border-outline-variant border p-6 md:p-8"

  if (submitted) {
    return (
      <div className={cn(panelClassName, "text-center")}>
        <h3 className="font-display text-primary text-[28px] font-light italic">
          {m.villa_reviews_form_success_title()}
        </h3>
        <p className="text-on-surface-variant mx-auto mt-3 max-w-[42ch] text-[15px] leading-relaxed">
          {m.villa_reviews_form_success_body()}
        </p>
      </div>
    )
  }

  return (
    <div className={panelClassName}>
      <span className="text-secondary inline-flex items-center gap-3 font-mono text-[11px] tracking-[0.22em] uppercase before:block before:h-px before:w-6 before:bg-current">
        {m.villa_reviews_form_chapter()}
      </span>
      <h3 className="font-display text-primary mt-3 text-[26px] leading-none font-light md:text-[32px]">
        <em className="italic-display">{m.villa_reviews_form_title()}</em>
      </h3>
      <p className="text-on-surface-variant mt-3 max-w-[52ch] text-[14.5px] leading-relaxed">
        {m.villa_reviews_form_intro()}
      </p>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-7">
        <div className="border-outline-variant border-b pb-5">
          <span className={fieldLabelClassName}>{m.villa_reviews_form_rating_label()}</span>
          <div className="mt-2">
            <StarPicker
              value={rating}
              size="lg"
              describeStar={(count) => m.villa_reviews_form_aria_rate_overall({ count })}
              onChange={handleRating}
            />
          </div>
        </div>

        <div className="border-outline-variant grid border-b">
          <label className="block py-4">
            <span className={fieldLabelClassName}>{m.villa_reviews_form_name_label()}</span>
            <Input
              type="text"
              autoComplete="name"
              aria-invalid={errors.authorName ? true : undefined}
              placeholder={m.villa_reviews_form_name_placeholder()}
              className={inputClassName}
              {...register("authorName", { required: true, validate: (v) => v.trim().length > 0 })}
            />
          </label>
        </div>

        <div className="border-outline-variant grid border-b">
          <label className="block py-4">
            <span className={fieldLabelClassName}>{m.villa_reviews_form_body_label()}</span>
            <Textarea
              rows={4}
              maxLength={2000}
              aria-invalid={errors.body ? true : undefined}
              placeholder={m.villa_reviews_form_body_placeholder()}
              className="text-primary placeholder:text-on-surface-variant/50 mt-1 min-h-[84px] w-full resize-none rounded-none border-0 bg-transparent px-0 py-0 text-[15px] leading-relaxed shadow-none focus-visible:border-0 focus-visible:ring-0 md:text-[15px]"
              {...register("body", { required: true, validate: (v) => v.trim().length > 0 })}
            />
          </label>
        </div>

        <fieldset className="border-outline-variant border-b py-5">
          <legend className={cn(fieldLabelClassName, "mb-1")}>
            {m.villa_reviews_form_categories_label()}
          </legend>
          <p className="text-on-surface-variant/70 mb-3 text-[13px]">
            {m.villa_reviews_form_categories_hint()}
          </p>
          <div className="grid max-w-[400px] gap-2.5">
            {CATEGORY_KEYS.map((key, i) => {
              const label = barLabels[i] ?? key
              return (
                <div key={key} className="flex items-center justify-between gap-4">
                  <span className="text-on-surface-variant font-mono text-[10.5px] tracking-[0.12em] uppercase">
                    {label}
                  </span>
                  <StarPicker
                    value={categories[key]}
                    size="sm"
                    describeStar={(count) =>
                      m.villa_reviews_form_aria_rate({ category: label, count })
                    }
                    onChange={(next) => handleCategory(key, next)}
                  />
                </div>
              )
            })}
          </div>
        </fieldset>

        {(errors.authorName || errors.body) && (
          <p role="alert" className="text-error mt-4 text-[13px]">
            {m.villa_reviews_form_required()}
          </p>
        )}

        {topLevelError && (
          <p
            role="alert"
            className="border-error/30 bg-error-container/20 text-error mt-4 border px-3 py-2 text-[13px]"
          >
            {topLevelError}
          </p>
        )}

        <Button
          type="submit"
          disabled={isPending}
          className="bg-primary text-on-primary hover:bg-primary-container mt-6 inline-flex h-auto w-full items-center justify-center gap-3 rounded-none px-6 py-[18px] font-mono text-[11px] tracking-[0.28em] uppercase disabled:opacity-60 md:w-auto"
        >
          {isPending ? m.villa_reviews_form_sending() : m.villa_reviews_form_submit()}
          {!isPending && <ArrowRight size={12} />}
        </Button>
      </form>
    </div>
  )
}
