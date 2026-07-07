import { createFileRoute } from "@tanstack/react-router"

import VillaDetailPage from "@/pages/villa-detail-page"

export const Route = createFileRoute("/_public/villa/$villaId")({
  component: VillaDetailPage,
})
