import { ApiError, getListBookingsQueryKey, useCreateBooking } from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
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
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useToast } from "@/components/ui/toast"

interface AddReservationFormValues {
  guestName: string
  guestEmail: string
  guestPhone: string
  checkIn: string
  checkOut: string
  adults: number
  source: "direct" | "airbnb" | "booking_com"
}

interface AddReservationDialogProps {
  property: "casadana" | "casacasay"
}

export default function AddReservationDialog({ property }: AddReservationDialogProps) {
  const [open, setOpen] = useState(false)
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<AddReservationFormValues>({
    defaultValues: {
      guestName: "",
      guestEmail: "",
      guestPhone: "",
      checkIn: "",
      checkOut: "",
      adults: 2,
      source: "direct",
    },
  })

  // Invalidate at the endpoint prefix (not the exact page/limit key) so this
  // also refreshes the stat row's separate limit:1 queries, not just this page.
  const bookingsQueryKey = getListBookingsQueryKey()

  const { mutate: createBooking, isPending } = useCreateBooking({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: bookingsQueryKey })
        toast("Réservation ajoutée")
        setOpen(false)
        reset()
      },
      onError: (err) => {
        if (err instanceof ApiError && err.code === "DATES_CONFLICT") {
          setError("checkOut", {
            type: "conflict",
            message: "Ces dates chevauchent une réservation existante.",
          })
        } else {
          toast("Impossible d'ajouter la réservation")
        }
      },
    },
  })

  const onSubmit = (values: AddReservationFormValues) => {
    createBooking({
      data: {
        villa_slug: property,
        guest_name: values.guestName,
        guest_email: values.guestEmail,
        guest_phone: values.guestPhone,
        check_in: values.checkIn,
        check_out: values.checkOut,
        adults: Number(values.adults),
        children: 0,
        source: values.source,
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button type="button" />}>Ajouter une réservation</DialogTrigger>
      <DialogContent>
        <DialogTitle>Ajouter une réservation</DialogTitle>
        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-3">
          <Field>
            <FieldLabel htmlFor="guestName">Nom du voyageur</FieldLabel>
            <Input id="guestName" {...register("guestName", { required: true })} />
          </Field>
          <Field>
            <FieldLabel htmlFor="guestEmail">E-mail</FieldLabel>
            <Input id="guestEmail" type="email" {...register("guestEmail", { required: true })} />
          </Field>
          <Field>
            <FieldLabel htmlFor="guestPhone">Téléphone</FieldLabel>
            <Input id="guestPhone" {...register("guestPhone", { required: true })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field>
              <FieldLabel htmlFor="checkIn">Arrivée</FieldLabel>
              <Input id="checkIn" type="date" {...register("checkIn", { required: true })} />
            </Field>
            <Field>
              <FieldLabel htmlFor="checkOut">Départ</FieldLabel>
              <Input id="checkOut" type="date" {...register("checkOut", { required: true })} />
              <FieldError errors={[errors.checkOut]} />
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Field>
              <FieldLabel htmlFor="adults">Voyageurs</FieldLabel>
              <Input
                id="adults"
                type="number"
                min={1}
                max={10}
                {...register("adults", { required: true, valueAsNumber: true })}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="source">Source</FieldLabel>
              <select
                id="source"
                {...register("source")}
                className="border-outline-variant text-on-surface h-8 rounded-lg border bg-transparent px-2.5 text-sm"
              >
                <option value="direct">Direct</option>
                <option value="airbnb">Airbnb</option>
                <option value="booking_com">Booking.com</option>
              </select>
            </Field>
          </div>
          <div className="mt-2 flex justify-end gap-2.5">
            <DialogClose render={<Button type="button" variant="outline" />}>Annuler</DialogClose>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Enregistrement…" : "Enregistrer la réservation"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
