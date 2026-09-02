import { useAdminLogout } from "@casa-dana/api"
import { useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { CalendarDays, History, LogOut, Star, Tag, UserCog } from "lucide-react"

const NAV_ITEMS = [
  { to: "/admin/reservations", label: "Réservations", icon: CalendarDays },
  { to: "/admin/pricing", label: "Tarifs", icon: Tag },
  { to: "/admin/reviews", label: "Avis", icon: Star },
  { to: "/admin/owner", label: "Propriétaire et accès", icon: UserCog },
  { to: "/admin/history", label: "Historique", icon: History },
] as const

export default function AdminSidebar() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { mutate: logout } = useAdminLogout({
    mutation: {
      onSuccess: () => {
        queryClient.clear()
        navigate({ to: "/admin/login" })
      },
    },
  })

  const handleLogout = () => logout()

  return (
    <aside className="bg-primary text-on-primary flex h-screen flex-col gap-6 p-4">
      <div className="px-2">
        <p className="text-sm font-bold">Casa Admin</p>
        <p className="text-on-primary/60 mt-1 font-mono text-[9.5px] tracking-[0.2em] uppercase">
          Interne · non public
        </p>
      </div>

      <nav className="flex flex-1 flex-col gap-0.5">
        {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
          <Link
            key={to}
            to={to}
            className="text-on-primary/75 hover:bg-primary-container hover:text-on-primary flex items-center gap-2.5 rounded-md px-3 py-2.5 text-[13.5px] font-medium"
            activeProps={{ className: "bg-white text-primary hover:bg-white hover:text-primary" }}
          >
            <Icon className="size-4 shrink-0" />
            {label}
          </Link>
        ))}
      </nav>

      <div className="flex flex-col gap-1.5">
        <a
          href="/"
          target="_blank"
          rel="noopener noreferrer"
          className="text-on-primary/65 hover:text-on-primary rounded-md px-2 py-1.5 text-[12.5px]"
        >
          ↗ Voir le site public
        </a>
        <button
          type="button"
          onClick={handleLogout}
          className="text-on-primary/65 hover:text-on-primary flex items-center gap-1.5 rounded-md px-2 py-1.5 text-left text-[12.5px]"
        >
          <LogOut className="size-3.5" />
          Se déconnecter
        </button>
      </div>
    </aside>
  )
}
