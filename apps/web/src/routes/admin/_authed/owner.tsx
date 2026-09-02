import { createFileRoute } from "@tanstack/react-router"

import AdminUsersTable from "@/components/admin/admin-users-table"
import CreateAdminUserDialog from "@/components/admin/create-admin-user-dialog"

export const Route = createFileRoute("/admin/_authed/owner")({
  component: OwnerPage,
})

function OwnerPage() {
  return (
    <div>
      <div className="mb-7 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h2 className="text-on-surface text-2xl font-bold">Propriétaire et accès</h2>
          <p className="text-on-surface-variant mt-1 text-[13.5px]">
            Comptes administrateurs ayant accès aux deux propriétés. Les coordonnées de l'hôte et
            les informations de paiement ne sont pas encore gérées ici.
          </p>
        </div>
        <CreateAdminUserDialog />
      </div>

      <div className="border-outline-variant bg-surface rounded-lg border">
        <div className="border-outline-variant border-b px-5 py-4">
          <h3 className="text-on-surface text-[14.5px] font-semibold">Administrateurs</h3>
        </div>
        <AdminUsersTable />
      </div>
    </div>
  )
}
