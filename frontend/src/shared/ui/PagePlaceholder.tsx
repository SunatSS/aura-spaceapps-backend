type Props = {
  title: string
  description: string
}

/** Temporary page body until the feature is implemented. Unique display title, top-left (App. F F.4.2). */
export function PagePlaceholder({ title, description }: Props) {
  return (
    <section>
      <h1 className="mb-3 text-[40px] leading-[48px] font-semibold tracking-[-0.01em]">{title}</h1>
      <p className="max-w-2xl text-ink-muted">{description}</p>
    </section>
  )
}
