type Props = {
  title: string
  description: string
}

/** Temporary page body until the feature is implemented. */
export function PagePlaceholder({ title, description }: Props) {
  return (
    <section>
      <h1 className="mb-3 font-display text-[56px] leading-[0.95] font-bold tracking-[-0.01em] uppercase">{title}</h1>
      <p className="max-w-2xl text-ink-muted">{description}</p>
    </section>
  )
}
