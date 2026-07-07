import { createFileRoute } from "@tanstack/react-router"

import ComingSoon from "@/components/admin/coming-soon"

export const Route = createFileRoute("/admin/_authed/history")({
  component: () => <ComingSoon title="History" />,
})
