import { render, screen, fireEvent } from '@testing-library/react'
import EditableTable from './EditableTable'

// ── Fixtures ──────────────────────────────────────────────────

const MEMBERS = {
  m1: { id: 'm1', name: 'Alice', color: '#aaa' },
  m2: { id: 'm2', name: 'Bob', color: '#bbb' },
}

const memberOf = id => MEMBERS[id] || null

const ROWS = [
  { id: 'r1', name: 'Fund A', invested: 50000, current: 60000, member: 'm1' },
  { id: 'r2', name: 'Fund B', invested: 30000, current: 35000, member: 'm2' },
]

const COLUMNS = [
  { key: 'name',     label: 'Fund',          align: 'left' },
  { key: 'member',   label: 'Owner',         isMember: true },
  { key: 'invested', label: 'Invested',      align: 'right', editable: true,  format: v => `₹${v}` },
  { key: 'current',  label: 'Current value', align: 'right', editable: true,  format: v => `₹${v}` },
]

// ── Tests ─────────────────────────────────────────────────────

describe('EditableTable', () => {
  describe('basic rendering', () => {
    it('renders the correct number of rows', () => {
      const { container } = render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      const rows = container.querySelectorAll('tbody tr')
      expect(rows).toHaveLength(2)
    })

    it('renders column headers', () => {
      render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      expect(screen.getByText('Fund')).toBeInTheDocument()
      expect(screen.getByText('Invested')).toBeInTheDocument()
      expect(screen.getByText('Current value')).toBeInTheDocument()
    })

    it('renders plain column values', () => {
      render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      expect(screen.getByText('Fund A')).toBeInTheDocument()
      expect(screen.getByText('Fund B')).toBeInTheDocument()
    })
  })

  describe('member column (showOwner)', () => {
    it('hides member column when showOwner=false', () => {
      render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      expect(screen.queryByText('Owner')).not.toBeInTheDocument()
    })

    it('shows member column when showOwner=true', () => {
      render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={true} memberOf={memberOf} />
      )
      expect(screen.getByText('Owner')).toBeInTheDocument()
    })
  })

  describe('editable cells', () => {
    it('renders EditCell for editable columns with formatted display', () => {
      render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      // Editable cells show formatted values as buttons
      expect(screen.getByText('₹50000')).toBeInTheDocument()
      expect(screen.getByText('₹60000')).toBeInTheDocument()
    })

    it('calls onMark with correct args when EditCell value changes', () => {
      const onMark = vi.fn()
      render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={onMark} showOwner={false} />
      )
      // Click the EditCell button for r1/invested to enter editing mode
      const investedBtn = screen.getByText('₹50000')
      fireEvent.click(investedBtn)

      // Now an input should appear — change its value and blur
      const input = screen.getByDisplayValue('50000')
      fireEvent.change(input, { target: { value: '55000' } })
      fireEvent.blur(input)

      expect(onMark).toHaveBeenCalledWith('r1', 'invested', 55000)
    })

    it('uses dirty value over row value in editable cells', () => {
      const dirty = { r1: { invested: 99999 } }
      render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={dirty} onMark={() => {}} showOwner={false} />
      )
      expect(screen.getByText('₹99999')).toBeInTheDocument()
    })
  })

  describe('totals bar', () => {
    it('does not render TotalsBar when totals prop is not provided', () => {
      const { container } = render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      expect(container.querySelector('.totals-bar')).not.toBeInTheDocument()
    })

    it('renders TotalsBar when totals prop is provided', () => {
      const totals = {
        label: '2 funds',
        invFn: row => row.invested,
        curFn: row => row.current,
      }
      const { container } = render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} totals={totals} />
      )
      expect(container.querySelector('.totals-bar')).toBeInTheDocument()
    })

    it('renders TotalsBar with the label from totals prop', () => {
      const totals = {
        label: '2 funds',
        invFn: row => row.invested,
        curFn: row => row.current,
      }
      render(
        <EditableTable rows={ROWS} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} totals={totals} />
      )
      expect(screen.getByText('2 funds')).toBeInTheDocument()
    })
  })

  describe('empty state', () => {
    it('shows default empty message when rows is empty', () => {
      render(
        <EditableTable rows={[]} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      expect(screen.getByText('No entries yet.')).toBeInTheDocument()
    })

    it('shows custom emptyMessage when rows is empty', () => {
      render(
        <EditableTable
          rows={[]}
          columns={COLUMNS}
          dirty={{}}
          onMark={() => {}}
          showOwner={false}
          emptyMessage="No mutual funds recorded yet."
        />
      )
      expect(screen.getByText('No mutual funds recorded yet.')).toBeInTheDocument()
    })

    it('does not render a table when rows is empty', () => {
      const { container } = render(
        <EditableTable rows={[]} columns={COLUMNS} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      expect(container.querySelector('table')).not.toBeInTheDocument()
    })
  })

  describe('custom render column', () => {
    it('renders custom render function output', () => {
      const columnsWithRender = [
        { key: 'name', label: 'Fund', align: 'left' },
        { key: 'status', label: 'Type', render: row => <span className="pill">{row.name}-pill</span> },
      ]
      render(
        <EditableTable rows={ROWS} columns={columnsWithRender} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      expect(screen.getByText('Fund A-pill')).toBeInTheDocument()
      expect(screen.getByText('Fund B-pill')).toBeInTheDocument()
    })
  })

  describe('computed columns', () => {
    it('renders formatted computed column values', () => {
      const columnsWithCompute = [
        { key: 'name', label: 'Stock', align: 'left' },
        { key: 'computed', label: 'Value', align: 'right', compute: row => row.invested * 2, format: v => `₹${v}` },
      ]
      render(
        <EditableTable rows={ROWS} columns={columnsWithCompute} dirty={{}} onMark={() => {}} showOwner={false} />
      )
      // 50000 * 2 = 100000
      expect(screen.getByText('₹100000')).toBeInTheDocument()
      // 30000 * 2 = 60000
      expect(screen.getByText('₹60000')).toBeInTheDocument()
    })
  })
})
