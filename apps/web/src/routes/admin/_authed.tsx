import { Outlet, createFileRoute, redirect } from "@tanstack/react-router"

import { getAdminMeQueryOptions } from "@casa-dana/api"
import AdminSidebar from "@/components/admin/admin-sidebar"
import { ToastProvider } from "@/components/ui/toast"

export const Route = createFileRoute("/admin/_authed")({
  beforeLoad: async ({ context }) => {
    try {
      await context.queryClient.fetchQuery(getAdminMeQueryOptions())
    } catch {
      throw redirect({ to: "/admin/login" })
    }
  },
  component: AuthedAdminLayout,
})

function AuthedAdminLayout() {
  return (
    <ToastProvider>
      <div className="grid min-h-screen grid-cols-[236px_1fr]">
        <AdminSidebar />
        <main className="px-10 py-8">
          <Outlet />
        </main>
      </div>
    </ToastProvider>
  )
}
