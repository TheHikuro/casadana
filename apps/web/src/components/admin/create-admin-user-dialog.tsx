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
        toast("Admin added")
        setOpen(false)
        reset()
      },
      onError: (err) => {
        if (err instanceof ApiError && err.code === "EMAIL_TAKEN") {
          setError("email", { type: "taken", message: "An admin with this email already exists." })
        } else {
          toast("Could not add admin")
        }
      },
    },
  })

  const onSubmit = (values: CreateAdminUserFormValues) => {
    createUser({ data: values })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button type="button" />}>Add admin</DialogTrigger>
      <DialogContent>
        <DialogTitle>Add admin</DialogTitle>
        <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-3">
          <Field>
            <FieldLabel htmlFor="newAdminEmail">Email</FieldLabel>
            <Input id="newAdminEmail" type="email" {...register("email", { required: true })} />
            <FieldError errors={[errors.email]} />
          </Field>
          <Field>
            <FieldLabel htmlFor="newAdminPassword">Password</FieldLabel>
            <Input
              id="newAdminPassword"
              type="password"
              {...register("password", {
                required: true,
                minLength: { value: 8, message: "Password must be at least 8 characters." },
              })}
            />
            <FieldError errors={[errors.password]} />
          </Field>
          <div className="mt-2 flex justify-end gap-2.5">
            <DialogClose render={<Button type="button" variant="outline" />}>Cancel</DialogClose>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Adding…" : "Add admin"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
