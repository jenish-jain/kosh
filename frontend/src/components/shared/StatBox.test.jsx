import { render, screen } from '@testing-library/react'
import StatBox from './StatBox'

describe('StatBox', () => {
  it('renders the label', () => {
    render(<StatBox label="Total net worth" value="₹12,00,000" />)
    expect(screen.getByText('Total net worth')).toBeInTheDocument()
  })

  it('renders the value', () => {
    render(<StatBox label="Monthly outflow" value="₹5,000" />)
    expect(screen.getByText('₹5,000')).toBeInTheDocument()
  })

  it('renders the subtitle when provided', () => {
    render(<StatBox label="Active SIPs" value="4 / 5" subtitle="next on 5 Jun" />)
    expect(screen.getByText('next on 5 Jun')).toBeInTheDocument()
  })

  it('does not render subtitle when not provided', () => {
    const { container } = render(<StatBox label="Label" value="Value" />)
    // Only label and value divs, no subtitle
    const divs = container.querySelectorAll('div')
    // Should have 2 divs (label + value container inner), but NOT a subtitle
    const texts = Array.from(divs).map(d => d.textContent)
    expect(texts).not.toContain('undefined')
  })

  it('applies custom style to the container', () => {
    const { container } = render(
      <StatBox label="Label" value="Val" style={{ paddingLeft: 24 }} />
    )
    const root = container.firstChild
    expect(root).toHaveStyle({ paddingLeft: '24px' })
  })
})
