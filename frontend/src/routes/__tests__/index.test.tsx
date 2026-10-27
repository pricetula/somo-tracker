import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Home } from '../index'

describe('Home component', () => {
  it('renders the home page heading', () => {
    render(<Home />)

    expect(screen.getByText('Somotracker')).toBeInTheDocument()
    expect(screen.getByText('TanStack Start scaffold ready.')).toBeInTheDocument()
  })
})