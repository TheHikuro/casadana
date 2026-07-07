import { Outlet, createFileRoute } from "@tanstack/react-router"

import Footer from "@/components/footer/footer"
import Navbar from "@/components/header/navbar"

export const Route = createFileRoute("/_public")({
  component: PublicLayout,
})

function PublicLayout() {
  return (
    <div className="flex min-h-screen flex-col">
      <Navbar />
      <main className="grow">
        <Outlet />
      </main>
      <Footer />
    </div>
  )
}
