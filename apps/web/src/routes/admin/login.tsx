import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router"
import { useForm } from "react-hook-form"

import { ApiError, getAdminMeQueryOptions, useAdminLogin } from "@casa-dana/api"
import { Button } from "@/components/ui/button"
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

export const Route = createFileRoute("/admin/login")({
  beforeLoad: async ({ context }) => {
    const alreadyAuthed = await context.queryClient
      .fetchQuery(getAdminMeQueryOptions())
      .then(() => true)
      .catch(() => false)
    if (alreadyAuthed) {
      throw redirect({ to: "/admin/reservations", search: { property: "casadana", page: 1 } })
    }
  },
  component: AdminLoginPage,
})

interface LoginFormValues {
  email: string
  password: string
}

function AdminLoginPage() {
  const navigate = useNavigate()
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<LoginFormValues>({
    defaultValues: { email: "", password: "" },
  })

  const { mutate: login, isPending } = useAdminLogin({
    mutation: {
      onSuccess: () => {
        navigate({ to: "/admin/reservations", search: { property: "casadana", page: 1 } })
      },
      onError: (err) => {
        const message =
          err instanceof ApiError && err.code === "INVALID_CREDENTIALS"
            ? "Incorrect email or password."
            : "Something went wrong. Try again."
        setError("password", { type: "invalid", message })
      },
    },
  })

  const onSubmit = (values: LoginFormValues) => {
    login({ data: values })
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-primary px-4">
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="w-full max-w-sm rounded-xl bg-surface p-10 shadow-editorial"
      >
        <p className="mb-1.5 font-mono text-[11px] tracking-[0.22em] text-on-surface-variant uppercase">
          Casa DaNa &amp; CasAy
        </p>
        <h1 className="mb-6 text-xl font-bold text-on-surface">Admin access</h1>

        <Field className="mb-3">
          <FieldLabel htmlFor="email">Email</FieldLabel>
          <Input id="email" type="email" autoComplete="username" {...register("email", { required: true })} />
        </Field>
        <Field className="mb-1">
          <FieldLabel htmlFor="password">Password</FieldLabel>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            {...register("password", { required: true })}
          />
        </Field>
        <FieldError errors={[errors.password]} className="mb-3 min-h-4" />

        <Button type="submit" disabled={isPending} className="w-full justify-center">
          {isPending ? "Signing in…" : "Enter dashboard"}
        </Button>
      </form>
    </div>
  )
}
