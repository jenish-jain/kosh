import { render, screen } from '@testing-library/react'
import { Icon } from './Icons'

describe('Icon', () => {
  it('should render an SVG element', () => {
    const { container } = render(<Icon name="plus" />)
    const svg = container.querySelector('svg')
    expect(svg).toBeInTheDocument()
  })

  it('should have correct default width and height of 18', () => {
    const { container } = render(<Icon name="plus" />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveAttribute('width', '18')
    expect(svg).toHaveAttribute('height', '18')
  })

  it('should reflect a custom size prop', () => {
    const { container } = render(<Icon name="plus" size={32} />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveAttribute('width', '32')
    expect(svg).toHaveAttribute('height', '32')
  })

  it('should render known icon names without throwing', () => {
    const names = ['plus', 'edit', 'upload', 'trash', 'x']
    for (const name of names) {
      const { container, unmount } = render(<Icon name={name} />)
      expect(container.querySelector('svg')).toBeInTheDocument()
      unmount()
    }
  })

  it('should apply a custom className', () => {
    const { container } = render(<Icon name="plus" className="my-icon" />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveClass('my-icon')
  })

  it('should apply a custom style', () => {
    const { container } = render(<Icon name="plus" style={{ color: 'red' }} />)
    const svg = container.querySelector('svg')
    // jsdom normalises colour keywords to rgb() values
    expect(svg).toHaveStyle({ color: 'rgb(255, 0, 0)' })
  })

  it('should respect the sw prop for stroke-width', () => {
    const { container } = render(<Icon name="plus" sw={3} />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveAttribute('stroke-width', '3')
  })
})
