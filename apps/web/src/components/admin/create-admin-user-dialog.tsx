import { ApiError, getListAdminUsersQueryKey, useCreateAdminUser } from "@casa-dana/api"
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

interface CreateAdminUserFormValues {
  email: string
  password: string
}

export default function CreateAdminUserDialog() {
  const [open, setOpen] = useState(false)
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<CreateAdminUserFormValues>({
    defaultValues: { email: "", password: "" },
  })

  const { mutate: createUser, isPending } = useCreateAdminUser({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: getListAdminUsersQueryKey() })
        toast("Administrateur ajouté")
        setOpen(false)
        reset()
      },
      onError: (err) => {
        if (err instanceof ApiError && err.code === "EMAIL_TAKEN") {
          setError("email", {
            type: "taken",
            message: "Un administrateur avec cet e-mail existe déjà.",
          })
        } else {
          toast("Impossible d'ajouter l'administrateur")
        }
      },
    },
  })

  const onSubmit = (values: CreateAdminUserFormValues) => {
    createUser({ data: values })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button type="button" />}>Ajouter un administrateur</DialogTrigger>
      <DialogContent>
        <DialogTitle>Ajouter un administrateur</DialogTitle>
        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-3">
          <Field>
            <FieldLabel htmlFor="newAdminEmail">E-mail</FieldLabel>
            <Input id="newAdminEmail" type="email" {...register("email", { required: true })} />
            <FieldError errors={[errors.email]} />
          </Field>
          <Field>
            <FieldLabel htmlFor="newAdminPassword">Mot de passe</FieldLabel>
            <Input
              id="newAdminPassword"
              type="password"
              {...register("password", {
                required: true,
                minLength: {
                  value: 8,
                  message: "Le mot de passe doit contenir au moins 8 caractères.",
                },
              })}
            />
            <FieldError errors={[errors.password]} />
          </Field>
          <div className="mt-2 flex justify-end gap-2.5">
            <DialogClose render={<Button type="button" variant="outline" />}>Annuler</DialogClose>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Ajout…" : "Ajouter un administrateur"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
