interface ComingSoonProps {
  title: string
}

export default function ComingSoon({ title }: ComingSoonProps) {
  return (
    <div>
      <h2 className="text-2xl font-bold text-on-surface">{title}</h2>
      <div className="mt-6 rounded-lg border border-outline-variant bg-surface px-5 py-16 text-center text-[13.5px] text-on-surface-variant">
        Coming soon.
      </div>
    </div>
  )
}
