import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MarketingLayout from './MarketingLayout.jsx'
import { resetServerModeForTests } from '../../data/serverMode.js'

function serverMode(mode) {
  resetServerModeForTests()
  globalThis.fetch = vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve({ mode }) }))
}

const renderLayout = () => render(<MemoryRouter><MarketingLayout><p>body</p></MarketingLayout></MemoryRouter>)

describe('MarketingLayout nav', () => {
  it('hides Pricing on a self-hosted deploy', async () => {
    serverMode('selfhost')
    renderLayout()
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledWith('/api/auth/config'))
    await screen.findByText('Features')
    await new Promise(r => setTimeout(r, 0))
    expect(screen.queryByText('Pricing')).toBeNull()
  })

  it('shows Pricing on hosted Kosh', async () => {
    serverMode('hosted')
    renderLayout()
    expect(await screen.findByText('Pricing')).toBeTruthy()
  })
})
