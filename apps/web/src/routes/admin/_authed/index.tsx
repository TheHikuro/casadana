import { createFileRoute, redirect } from "@tanstack/react-router"

export const Route = createFileRoute("/admin/_authed/")({
  beforeLoad: () => {
    throw redirect({ to: "/admin/reservations", search: { property: "casadana", page: 1 } })
  },
})
