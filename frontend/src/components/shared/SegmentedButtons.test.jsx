import { render, screen, fireEvent } from '@testing-library/react'
import SegmentedButtons from './SegmentedButtons'

const OPTIONS = [
  { value: 'mf', label: 'Mutual Fund' },
  { value: 'stocks', label: 'Stocks' },
  { value: 'metals', label: 'Metals' },
]

describe('SegmentedButtons', () => {
  it('renders all options', () => {
    render(<SegmentedButtons options={OPTIONS} value="mf" onChange={() => {}} />)
    expect(screen.getByText('Mutual Fund')).toBeInTheDocument()
    expect(screen.getByText('Stocks')).toBeInTheDocument()
    expect(screen.getByText('Metals')).toBeInTheDocument()
  })

  it('active option has "active" class', () => {
    render(<SegmentedButtons options={OPTIONS} value="stocks" onChange={() => {}} />)
    const buttons = screen.getAllByRole('button')
    const activeBtn = buttons.find(b => b.textContent === 'Stocks')
    expect(activeBtn).toHaveClass('active')
  })

  it('non-active options do not have "active" class', () => {
    render(<SegmentedButtons options={OPTIONS} value="mf" onChange={() => {}} />)
    const buttons = screen.getAllByRole('button')
    const stocksBtn = buttons.find(b => b.textContent === 'Stocks')
    expect(stocksBtn).not.toHaveClass('active')
  })

  it('calls onChange with the option value when clicked', () => {
    const onChange = vi.fn()
    render(<SegmentedButtons options={OPTIONS} value="mf" onChange={onChange} />)
    fireEvent.click(screen.getByText('Stocks'))
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith('stocks')
  })

  it('calls onChange with the correct value for each option', () => {
    const onChange = vi.fn()
    render(<SegmentedButtons options={OPTIONS} value="mf" onChange={onChange} />)
    fireEvent.click(screen.getByText('Metals'))
    expect(onChange).toHaveBeenCalledWith('metals')
  })

  it('renders with two options', () => {
    const opts = [{ value: 'FD', label: 'Fixed Deposit' }, { value: 'RD', label: 'Recurring Deposit' }]
    render(<SegmentedButtons options={opts} value="FD" onChange={() => {}} />)
    expect(screen.getAllByRole('button')).toHaveLength(2)
  })
})
