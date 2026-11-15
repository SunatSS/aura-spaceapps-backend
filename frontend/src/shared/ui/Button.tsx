import clsx from 'clsx'
import type { ButtonHTMLAttributes } from 'react'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  /** command = NASA App. F Table F.5-2 dark green (commandable element); secondary = outline. */
  variant?: 'command' | 'secondary'
}

export function Button({ variant = 'secondary', className, ...rest }: Props) {
  return (
    <button
      type="button"
      {...rest}
      className={clsx(
        'inline-flex h-10 items-center gap-2 rounded-md border-[1.5px] px-4 text-[15px] leading-5 font-semibold active:translate-y-px',
        'disabled:cursor-not-allowed disabled:border-dashed disabled:border-unavailable disabled:bg-transparent disabled:text-unavailable',
        variant === 'command' ? 'border-command-edge bg-command text-on-command' : 'border-line-strong text-ink',
        className,
      )}
    />
  )
}
