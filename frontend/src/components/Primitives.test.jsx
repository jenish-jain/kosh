import { render, screen, fireEvent, act } from '@testing-library/react'
import { Avatar, GainPill, Modal, Field, Toast, SaveBar } from './Primitives'

// ── Avatar ────────────────────────────────────────────────────

describe('Avatar', () => {
  it('should render the first initial of the member name', () => {
    render(<Avatar member={{ name: 'John Doe', color: '#ff0000' }} />)
    // Avatar takes first letter of each word (up to 2), uppercase
    expect(screen.getByText('JD')).toBeInTheDocument()
  })

  it('should apply member color as background style', () => {
    const { container } = render(
      <Avatar member={{ name: 'Alice', color: '#123456' }} />
    )
    const span = container.querySelector('.avatar')
    expect(span).toHaveStyle({ background: '#123456' })
  })
})

// ── GainPill ──────────────────────────────────────────────────

describe('GainPill', () => {
  it('should show positive percentage with + prefix', () => {
    render(<GainPill value={500} pct={4.2} />)
    expect(screen.getByText('+4.2%')).toBeInTheDocument()
  })

  it('should show negative percentage with - prefix', () => {
    render(<GainPill value={-200} pct={-3.5} />)
    expect(screen.getByText('-3.5%')).toBeInTheDocument()
  })
})

// ── Modal ─────────────────────────────────────────────────────

describe('Modal', () => {
  it('should render the title', () => {
    render(
      <Modal title="Test Modal" onClose={() => {}}>
        <p>Content</p>
      </Modal>
    )
    expect(screen.getByText('Test Modal')).toBeInTheDocument()
  })

  it('should render children inside the modal', () => {
    render(
      <Modal title="Modal" onClose={() => {}}>
        <span>Inner content</span>
      </Modal>
    )
    expect(screen.getByText('Inner content')).toBeInTheDocument()
  })

  it('should call onClose when the backdrop is clicked', () => {
    const onClose = vi.fn()
    const { container } = render(
      <Modal title="Modal" onClose={onClose}>
        <p>Child</p>
      </Modal>
    )
    const backdrop = container.querySelector('.modal-backdrop')
    fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('should call onClose when the × button is clicked', () => {
    const onClose = vi.fn()
    render(
      <Modal title="Modal" onClose={onClose}>
        <p>Child</p>
      </Modal>
    )
    // The close button is the ghost button in the modal header
    const closeButton = screen.getByRole('button')
    fireEvent.click(closeButton)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('should call onClose when Escape key is pressed', () => {
    const onClose = vi.fn()
    render(
      <Modal title="Modal" onClose={onClose}>
        <p>Child</p>
      </Modal>
    )
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})

// ── Field ─────────────────────────────────────────────────────

describe('Field', () => {
  it('should render the label text', () => {
    render(
      <Field label="Email">
        <input type="email" />
      </Field>
    )
    expect(screen.getByText('Email')).toBeInTheDocument()
  })

  it('should render children inside the field', () => {
    render(
      <Field label="Name">
        <input type="text" placeholder="Enter name" />
      </Field>
    )
    expect(screen.getByPlaceholderText('Enter name')).toBeInTheDocument()
  })
})

// ── Toast ─────────────────────────────────────────────────────

describe('Toast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should render the message', () => {
    render(<Toast message="Saved successfully" type="success" onDone={() => {}} />)
    expect(screen.getByText('Saved successfully')).toBeInTheDocument()
  })

  it('should call onDone after 3 seconds', () => {
    const onDone = vi.fn()
    render(<Toast message="Done" type="success" onDone={onDone} />)

    expect(onDone).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(3100)
    })

    expect(onDone).toHaveBeenCalledTimes(1)
  })
})

// ── SaveBar ───────────────────────────────────────────────────

describe('SaveBar', () => {
  it('should show the save button when dirty=true', () => {
    render(<SaveBar dirty={true} saving={false} onSave={() => {}} />)
    expect(screen.getByRole('button')).toBeInTheDocument()
    expect(screen.getByText('Save changes')).toBeInTheDocument()
  })

  it('should not render anything when dirty=false', () => {
    const { container } = render(
      <SaveBar dirty={false} saving={false} onSave={() => {}} />
    )
    expect(container.firstChild).toBeNull()
  })

  it('should call onSave when the save button is clicked', () => {
    const onSave = vi.fn()
    render(<SaveBar dirty={true} saving={false} onSave={onSave} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onSave).toHaveBeenCalledTimes(1)
  })

  it('should show "Saving…" text when saving=true', () => {
    render(<SaveBar dirty={true} saving={true} onSave={() => {}} />)
    expect(screen.getByText('Saving…')).toBeInTheDocument()
  })
})
