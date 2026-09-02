import {
  getListAdminUsersQueryKey,
  useAdminMe,
  useDeleteAdminUser,
  useListAdminUsers,
} from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
import { Trash2 } from "lucide-react"

import { useToast } from "@/components/ui/toast"

export default function AdminUsersTable() {
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const { data: me } = useAdminMe()
  const { data } = useListAdminUsers()
  const users = data?.users ?? []

  const { mutate: deleteUser } = useDeleteAdminUser({
    mutation: {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: getListAdminUsersQueryKey() })
        toast("Administrateur supprimé")
      },
      onError: () => toast("Impossible de supprimer l'administrateur"),
    },
  })

  const handleDelete = (id: string) => {
    if (window.confirm("Retirer l'accès de cet administrateur ?")) {
      deleteUser({ id })
    }
  }

  if (users.length === 0) {
    return (
      <div className="text-on-surface-variant px-5 py-10 text-center text-[13.5px]">
        Aucun compte administrateur pour le moment.
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[480px] border-collapse text-[13px]">
        <thead>
          <tr className="border-outline-variant bg-surface-container-low text-on-surface-variant border-b text-left text-[10.5px] font-semibold tracking-[0.08em] uppercase">
            <th className="px-5 py-2.5">E-mail</th>
            <th className="px-5 py-2.5">Ajouté le</th>
            <th className="px-5 py-2.5" />
          </tr>
        </thead>
        <tbody>
          {users.map((user) => {
            const isSelf = user.id === me?.id
            return (
              <tr key={user.id} className="border-outline-variant border-b last:border-0">
                <td className="text-on-surface px-5 py-3">
                  {user.email}
                  {isSelf && <span className="text-on-surface-variant ml-2">(vous)</span>}
                </td>
                <td className="text-on-surface-variant px-5 py-3 font-mono">
                  {user.created_at ? new Date(user.created_at).toLocaleDateString("fr-FR") : "—"}
                </td>
                <td className="px-5 py-3">
                  {!isSelf && (
                    <button
                      type="button"
                      onClick={() => user.id && handleDelete(user.id)}
                      aria-label="Supprimer l'administrateur"
                      className="text-on-surface-variant hover:bg-error-container hover:text-on-error-container rounded-md p-1.5"
                    >
                      <Trash2 className="size-3.5" />
                    </button>
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
