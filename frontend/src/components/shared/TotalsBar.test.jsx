import { render, screen } from '@testing-library/react'
import TotalsBar from './TotalsBar'

describe('TotalsBar', () => {
  it('renders the label', () => {
    render(<TotalsBar inv={100000} cur={120000} label="3 funds" />)
    expect(screen.getByText('3 funds')).toBeInTheDocument()
  })

  it('renders Total invested label', () => {
    render(<TotalsBar inv={100000} cur={120000} label="2 stocks" />)
    expect(screen.getByText('Total invested')).toBeInTheDocument()
  })

  it('renders formatted inv amount', () => {
    render(<TotalsBar inv={100000} cur={120000} label="1 fund" />)
    // fmtINR(100000) = ₹1,00,000
    expect(screen.getByText('₹1,00,000')).toBeInTheDocument()
  })

  it('renders formatted cur amount when cur > 0', () => {
    render(<TotalsBar inv={100000} cur={120000} label="1 fund" />)
    // fmtINR(120000) = ₹1,20,000
    expect(screen.getByText('₹1,20,000')).toBeInTheDocument()
  })

  it('renders — when cur is 0', () => {
    render(<TotalsBar inv={50000} cur={0} label="1 plan" />)
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('uses default curLabel "Current value"', () => {
    render(<TotalsBar inv={100000} cur={120000} label="test" />)
    expect(screen.getByText('Current value')).toBeInTheDocument()
  })

  it('uses custom curLabel when provided', () => {
    render(<TotalsBar inv={100000} cur={120000} label="test" curLabel="Total value" />)
    expect(screen.getByText('Total value')).toBeInTheDocument()
  })
})
