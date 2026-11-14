type Props = {
  title: string
  description: string
}

/** Temporary page body until the feature is implemented. */
export function PagePlaceholder({ title, description }: Props) {
  return (
    <section>
      <h1 className="mb-2 text-2xl font-semibold">{title}</h1>
      <p className="max-w-2xl text-slate-400">{description}</p>
    </section>
  )
}
