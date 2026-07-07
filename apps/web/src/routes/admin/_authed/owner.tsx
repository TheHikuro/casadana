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
          <h2 className="text-on-surface text-2xl font-bold">Owner & access</h2>
          <p className="text-on-surface-variant mt-1 text-[13.5px]">
            Admin accounts with access to both properties. Host contact and payout details aren't
            managed here yet.
          </p>
        </div>
        <CreateAdminUserDialog />
      </div>

      <div className="border-outline-variant bg-surface rounded-lg border">
        <div className="border-outline-variant border-b px-5 py-4">
          <h3 className="text-on-surface text-[14.5px] font-semibold">Admin users</h3>
        </div>
        <AdminUsersTable />
      </div>
    </div>
  )
}
