import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { Legend } from './components/Legend.tsx'

describe('Legend', () => {
  beforeEach(() => localStorage.clear())

  it('is collapsed by default and opens with the requested sections only', () => {
    render(<Legend sections={['classes', 'values']} />)
    const toggle = screen.getByRole('button', { name: /legend/i })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('Alert classes')).not.toBeInTheDocument()

    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Alert classes')).toBeInTheDocument()
    expect(screen.getByText('Values')).toBeInTheDocument()
    expect(screen.queryByText('Top bar')).not.toBeInTheDocument()
  })

  it('remembers that it was opened', () => {
    render(<Legend sections={['statusbar']} />)
    fireEvent.click(screen.getByRole('button', { name: /legend/i }))
    expect(localStorage.getItem('aura.legend.open')).toBe('1')
  })
})
