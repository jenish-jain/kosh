import { useState } from 'react'
import { useData } from '../../data/context.jsx'
import { holdingsFor, scope } from '../../data/aggregate.js'
import { todayDisplay } from '../../data/schedule.js'
import { KICK } from '../../data/tokens.js'
import { EdRule, SaveBar } from '../../components/Primitives.jsx'
import { Icon } from '../../components/Icons.jsx'
import UploadZone from '../../components/UploadZone.jsx'
import { requestDriveAccessToken } from '../../data/driveAuth.js'
import SummaryTiles from './SummaryTiles.jsx'
import MFTable from './tables/MFTable.jsx'
import StockTable from './tables/StockTable.jsx'
import MetalTable from './tables/MetalTable.jsx'
import FixedTable from './tables/FixedTable.jsx'
import NPSTable from './tables/NPSTable.jsx'
import InsuranceTable from './tables/InsuranceTable.jsx'
import AddModal from './AddModal.jsx'
import NPSImportModal from './NPSImportModal.jsx'
import { extractedToForm } from './formMappers.js'

export default function Investments({ data, memberId, showToast }) {
  const { update, clientId } = useData()
  const [tab, setTab] = useState('mf')
  const [showAdd, setShowAdd] = useState(false)
  const [showUpload, setShowUpload] = useState(false)
  const [uploadPreFill, setUploadPreFill] = useState(null)
  const [dirty, setDirty] = useState({})
  const [saving, setSaving] = useState(false)
  const [npsImportBatch, setNpsImportBatch] = useState(null)

  const uploadableTabs = ['fixed', 'insurance', 'metals', 'nps']
  const canUpload      = uploadableTabs.includes(tab)
  const uploadDocType  = { fixed: 'fd', insurance: 'insurance', metals: 'metals', nps: 'nps' }

  const handleExtracted = (fields, driveURL) => {
    // NPS statements yield a batch — show a confirm-and-bulk-import modal
    if (tab === 'nps' && Array.isArray(fields.holdings)) {
      setNpsImportBatch({
        pran: fields.pran || '',
        net_total_invested: Number(fields.net_total_invested) || 0,
        member: memberId || (data.members?.[0]?.id || 'you'),
        holdings: fields.holdings,
      })
      setShowUpload(false)
      return
    }
    const pre = extractedToForm(tab, fields)
    if (driveURL) pre.docLink = driveURL
    setUploadPreFill(pre)
    setShowUpload(false)
    setShowAdd(true)
  }

  const h   = holdingsFor(data, memberId)
  const all = !memberId
  const tabs = [
    ['mf',        'Mutual Funds',      h.mf.length],
    ['stocks',    'Stocks',            h.stocks.length],
    ['fixed',     'Deposits',          h.fixed.length],
    ['nps',       'NPS',               h.nps.length],
    ['metals',    'Gold & Silver',     h.metals.length],
    ['insurance', 'Insurance & Plans', h.insurance.length],
  ]

  const sheetMap = { mf: 'MF', stocks: 'Stocks', metals: 'Metals', insurance: 'Insurance', fixed: 'Fixed', nps: 'NPS' }

  const hasDirty = Object.keys(dirty).length > 0

  const save = async () => {
    setSaving(true)
    try {
      const promises = Object.entries(dirty).map(([id, patch]) =>
        update(sheetMap[tab], id, patch)
      )
      await Promise.all(promises)
      setDirty({})
      showToast('Saved to Google Sheet ✓')
    } catch (e) {
      showToast('Save failed — check connection', 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fade-in">
      {/* Header */}
      <div className="stmt-band">
        <div style={{ ...KICK, letterSpacing: '.18em' }}>Schedule of holdings</div>
        <div className="stmt-meta">{scope(data, memberId)} · As on {todayDisplay()}</div>
      </div>
      <EdRule thick />
      <SummaryTiles
        data={data}
        memberId={memberId}
        tab={tab}
        setTab={t => { setTab(t); setDirty({}) }}
      />
      <EdRule />

      {/* Tab bar + actions */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 18, flexWrap: 'wrap' }}>
        <div className="seg">
          {tabs.map(([k, label, n]) => (
            <button key={k} className={tab === k ? 'active' : ''} onClick={() => { setTab(k); setDirty({}) }}>
              {label} <span style={{ opacity: .55, fontWeight: 700 }}>{n}</span>
            </button>
          ))}
        </div>
        <div style={{ flex: 1 }} />
        {canUpload && (
          <button className="btn" onClick={() => {
            setUploadPreFill(null)
            setShowUpload(true)
            // Fired synchronously from this click so the browser treats the
            // resulting Google consent popup as user-initiated — kicking it
            // off from inside an async chain (e.g. on file drop) gets it
            // silently blocked. UploadZone awaits whatever this resolves to.
            requestDriveAccessToken(clientId).catch(() => {})
          }}>
            <Icon name="upload" size={15} /> Upload PDF
          </button>
        )}
        <button className="btn primary" onClick={() => { setUploadPreFill(null); setShowAdd(true) }}>
          <Icon name="plus" size={15} /> Add
        </button>
      </div>

      {/* Tables */}
      {tab === 'mf'        && <MFTable        data={data} rows={h.mf}        all={all} dirty={dirty} setDirty={setDirty} />}
      {tab === 'stocks'    && <StockTable     data={data} rows={h.stocks}    all={all} dirty={dirty} setDirty={setDirty} />}
      {tab === 'fixed'     && <FixedTable     data={data} rows={h.fixed}     all={all} />}
      {tab === 'nps'       && <NPSTable       data={data} rows={h.nps}       all={all} dirty={dirty} setDirty={setDirty} />}
      {tab === 'metals'    && <MetalTable     data={data} rows={h.metals}    all={all} dirty={dirty} setDirty={setDirty} />}
      {tab === 'insurance' && <InsuranceTable data={data} rows={h.insurance} all={all} dirty={dirty} setDirty={setDirty} />}

      <SaveBar dirty={hasDirty} saving={saving} onSave={save} />

      {showAdd && (
        <AddModal
          tab={tab} memberId={memberId} data={data}
          initialForm={uploadPreFill}
          onClose={() => { setShowAdd(false); setUploadPreFill(null) }}
        />
      )}
      {npsImportBatch && (
        <NPSImportModal
          batch={npsImportBatch}
          data={data}
          onDone={(count) => { setNpsImportBatch(null); showToast(`Imported ${count} NPS holding${count !== 1 ? 's' : ''} ✓`) }}
          onClose={() => setNpsImportBatch(null)}
        />
      )}
      {showUpload && (
        <UploadZone docType={uploadDocType[tab]} onExtracted={handleExtracted} onClose={() => setShowUpload(false)} />
      )}
    </div>
  )
}
