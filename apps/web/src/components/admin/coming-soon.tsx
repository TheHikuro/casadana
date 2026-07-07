interface ComingSoonProps {
  title: string
}

export default function ComingSoon({ title }: ComingSoonProps) {
  return (
    <div>
      <h2 className="text-on-surface text-2xl font-bold">{title}</h2>
      <div className="border-outline-variant bg-surface text-on-surface-variant mt-6 rounded-lg border px-5 py-16 text-center text-[13.5px]">
        Coming soon.
      </div>
    </div>
  )
}
