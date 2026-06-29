import { useState } from 'react'
import { SOAP, CIDSuggestion, EvolutionResponse } from '../services/api'
import { CidSuggestions } from './CidSuggestions'

interface SoapEditorProps {
  evolution: EvolutionResponse
  onSave: (patch: { soap?: Partial<SOAP>; cid_suggestions?: CIDSuggestion[]; status?: 'finalized' }) => Promise<void>
  onCancel: () => void
  isSaving?: boolean
}

export function SoapEditor({ evolution, onSave, onCancel, isSaving = false }: SoapEditorProps) {
  const [soap, setSoap] = useState<SOAP>({ ...evolution.soap })
  const [cids, setCids] = useState<CIDSuggestion[]>([...evolution.cid_suggestions])

  function updateField(field: keyof SOAP, value: string) {
    setSoap(prev => ({ ...prev, [field]: value }))
  }

  function removeCid(index: number) {
    setCids(prev => prev.filter((_, i) => i !== index))
  }

  async function handleSave(finalize = false) {
    await onSave({
      soap,
      cid_suggestions: cids,
      ...(finalize ? { status: 'finalized' } : {}),
    })
  }

  return (
    <div className="rounded-lg border border-gray-300 bg-white shadow-sm overflow-hidden">
      <header className="flex items-center justify-between px-4 py-3 border-b border-gray-200 bg-gray-50">
        <h2 className="font-semibold text-grafite-700">Editar Evolução SOAP</h2>
      </header>

      <div className="divide-y divide-gray-100">
        <EditorField
          label="S — Subjetivo"
          value={soap.s}
          onChange={v => updateField('s', v)}
          disabled={isSaving}
        />
        <EditorField
          label="O — Objetivo"
          value={soap.o}
          onChange={v => updateField('o', v)}
          disabled={isSaving}
        />
        <EditorField
          label="A — Avaliação"
          value={soap.a}
          onChange={v => updateField('a', v)}
          disabled={isSaving}
        />
        <EditorField
          label="P — Plano"
          value={soap.p}
          onChange={v => updateField('p', v)}
          disabled={isSaving}
        />
      </div>

      {cids.length > 0 && (
        <div className="px-4 py-3 border-t border-gray-200">
          <CidSuggestions suggestions={cids} onRemove={removeCid} />
        </div>
      )}

      <footer className="flex items-center justify-end gap-3 px-4 py-3 border-t border-gray-200 bg-gray-50">
        <button
          onClick={onCancel}
          disabled={isSaving}
          className="px-4 py-2 text-sm text-grafite-600 hover:text-grafite-800 disabled:opacity-50"
        >
          Cancelar
        </button>
        <button
          onClick={() => handleSave(false)}
          disabled={isSaving}
          className="px-4 py-2 text-sm rounded border border-navy-700 text-navy-700
                     hover:bg-navy-50 disabled:opacity-50 transition-colors"
        >
          {isSaving ? 'Salvando...' : 'Salvar rascunho'}
        </button>
        <button
          onClick={() => handleSave(true)}
          disabled={isSaving}
          className="px-4 py-2 text-sm rounded bg-navy-700 text-white
                     hover:bg-navy-600 disabled:opacity-50 transition-colors"
        >
          {isSaving ? 'Salvando...' : 'Finalizar'}
        </button>
      </footer>
    </div>
  )
}

interface EditorFieldProps {
  label: string
  value: string
  onChange: (value: string) => void
  disabled?: boolean
}

function EditorField({ label, value, onChange, disabled }: EditorFieldProps) {
  return (
    <div className="px-4 py-3">
      <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1">
        {label}
      </label>
      <textarea
        value={value}
        onChange={e => onChange(e.target.value)}
        disabled={disabled}
        rows={3}
        className="w-full text-sm text-grafite-600 border border-gray-300 rounded px-2 py-1.5
                   focus:outline-none focus:ring-2 focus:ring-navy-300 focus:border-navy-500
                   disabled:bg-gray-50 resize-y leading-relaxed"
        aria-label={label}
      />
    </div>
  )
}
