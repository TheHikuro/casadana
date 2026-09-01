import { Outlet, createFileRoute } from "@tanstack/react-router"

import Footer from "@/components/footer/footer"
import Navbar from "@/components/header/navbar"
import { ToastProvider } from "@/components/ui/toast"

export const Route = createFileRoute("/_public")({
  component: PublicLayout,
})

// The public tree gets its own ToastProvider (the admin tree has one on
// /admin/_authed): the review form on a villa page confirms a submission
// through a toast, and the two route trees never render at the same time.
function PublicLayout() {
  return (
    <ToastProvider>
      <div className="flex min-h-screen flex-col">
        <Navbar />
        <main className="grow">
          <Outlet />
        </main>
        <Footer />
      </div>
    </ToastProvider>
  )
}
