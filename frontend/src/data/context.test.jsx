import { renderHook, waitFor, act } from '@testing-library/react'
import { DataProvider, useData } from './context'
import * as api from '../data/api'

vi.mock('../data/api', () => ({
  fetchData: vi.fn(),
  getCached: vi.fn(),
  addRow: vi.fn(),
  updateRow: vi.fn(),
  deleteRow: vi.fn(),
}))

const wrapper = ({ children }) => (
  <DataProvider clientId="test-client-id">{children}</DataProvider>
)

beforeEach(() => {
  vi.clearAllMocks()
})

describe('useData', () => {
  it('should throw when called outside DataProvider', () => {
    // Suppress React's error boundary output for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    expect(() => renderHook(() => useData())).toThrow(
      'useData must be used inside DataProvider'
    )
    consoleSpy.mockRestore()
  })
})

describe('DataProvider', () => {
  it('should call fetchData on mount', async () => {
    api.fetchData.mockResolvedValue({ mf: [] })

    renderHook(() => useData(), { wrapper })

    await waitFor(() => {
      expect(api.fetchData).toHaveBeenCalledTimes(1)
    })
  })

  it('should expose loading=true initially then false after data loads', async () => {
    let resolveData
    api.fetchData.mockReturnValue(
      new Promise(resolve => { resolveData = resolve })
    )

    const { result } = renderHook(() => useData(), { wrapper })

    expect(result.current.loading).toBe(true)

    await act(async () => {
      resolveData({ mf: [] })
    })

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })
  })

  it('should expose fetched data after load', async () => {
    const payload = { mf: [{ id: '1', fund: 'HDFC' }], stocks: [] }
    api.fetchData.mockResolvedValue(payload)

    const { result } = renderHook(() => useData(), { wrapper })

    await waitFor(() => {
      expect(result.current.data).toEqual(payload)
    })
  })

  it('should expose error when fetchData throws', async () => {
    api.fetchData.mockRejectedValue(new Error('Network error'))

    const { result } = renderHook(() => useData(), { wrapper })

    await waitFor(() => {
      expect(result.current.error).toBe('Network error')
      expect(result.current.loading).toBe(false)
    })
  })

  it('should expose clientId through useData', async () => {
    api.fetchData.mockResolvedValue({})

    const { result } = renderHook(() => useData(), { wrapper })

    await waitFor(() => {
      expect(result.current.clientId).toBe('test-client-id')
    })
  })

  it('should call addRow and then refetch data', async () => {
    api.fetchData.mockResolvedValue({ mf: [] })
    api.addRow.mockResolvedValue({ id: 'new-row' })

    const { result } = renderHook(() => useData(), { wrapper })
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(async () => {
      await result.current.add('mf', { fund: 'SBI', amount: 500 })
    })

    expect(api.addRow).toHaveBeenCalledWith('mf', { fund: 'SBI', amount: 500 })
    expect(api.fetchData).toHaveBeenCalledTimes(2)
  })

  it('should call updateRow and then refetch data', async () => {
    api.fetchData.mockResolvedValue({ mf: [] })
    api.updateRow.mockResolvedValue({ id: 'row-1', fund: 'Updated' })

    const { result } = renderHook(() => useData(), { wrapper })
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(async () => {
      await result.current.update('mf', 'row-1', { fund: 'Updated' })
    })

    expect(api.updateRow).toHaveBeenCalledWith('mf', 'row-1', { fund: 'Updated' })
    expect(api.fetchData).toHaveBeenCalledTimes(2)
  })

  it('should call deleteRow and then refetch data', async () => {
    api.fetchData.mockResolvedValue({ mf: [] })
    api.deleteRow.mockResolvedValue({ deleted: true })

    const { result } = renderHook(() => useData(), { wrapper })
    await waitFor(() => expect(result.current.loading).toBe(false))

    await act(async () => {
      await result.current.remove('mf', 'row-1')
    })

    expect(api.deleteRow).toHaveBeenCalledWith('mf', 'row-1')
    expect(api.fetchData).toHaveBeenCalledTimes(2)
  })
})
