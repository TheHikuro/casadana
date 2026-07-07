import { createFileRoute } from "@tanstack/react-router"

import ComingSoon from "@/components/admin/coming-soon"

export const Route = createFileRoute("/admin/_authed/owner")({
  component: () => <ComingSoon title="Owner & access" />,
})
