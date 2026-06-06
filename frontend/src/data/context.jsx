import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { fetchData, addRow, updateRow, deleteRow } from './api.js'

const DataContext = createContext(null)

export function DataProvider({ children }) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const load = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const d = await fetchData()
      setData(d)
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  const mutate = useCallback(async (fn) => {
    await fn()
    await load() // refetch after mutation
  }, [load])

  const add = useCallback((sheet, row) =>
    mutate(() => addRow(sheet, row)), [mutate])

  const update = useCallback((sheet, id, patch) =>
    mutate(() => updateRow(sheet, id, patch)), [mutate])

  const remove = useCallback((sheet, id) =>
    mutate(() => deleteRow(sheet, id)), [mutate])

  return (
    <DataContext.Provider value={{ data, loading, error, reload: load, add, update, remove }}>
      {children}
    </DataContext.Provider>
  )
}

export function useData() {
  const ctx = useContext(DataContext)
  if (!ctx) throw new Error('useData must be used inside DataProvider')
  return ctx
}
