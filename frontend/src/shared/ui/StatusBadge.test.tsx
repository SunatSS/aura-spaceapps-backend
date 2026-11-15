import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { StatusBadge } from './StatusBadge.tsx'

describe('StatusBadge', () => {
  it('shows the NASA class word, never colour alone', () => {
    render(<StatusBadge alert="warning" />)
    expect(screen.getByText('Warning')).toBeInTheDocument()
  })

  it('uses on-alert ink when drawn on a warning fill', () => {
    render(<StatusBadge alert="warning" quiet onFill />)
    const badge = screen.getByText('Warning')
    expect(badge.className).toContain('text-on-alert')
    expect(badge.className).not.toContain('text-warning-ink')
  })
})
