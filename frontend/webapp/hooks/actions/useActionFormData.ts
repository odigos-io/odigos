import { useEffect, useState } from 'react'

type Signal = 'TRACES' | 'METRICS' | 'LOGS'
type FormData = {
  type: string
  name: string
  notes: string
  disable: boolean
  signals: Signal[]
  details: string
}

const INITIAL: FormData = {
  type: '',
  name: '',
  notes: '',
  disable: true,
  signals: [],
  details: '',
}

export function useActionFormData() {
  const [formData, setFormData] = useState({ ...INITIAL })
  const [exportedSignals, setExportedSignals] = useState({
    logs: false,
    metrics: false,
    traces: false,
  })

  const resetFormData = () => {
    setFormData({ ...INITIAL })
  }

  const handleFormChange = (key: keyof typeof INITIAL, val: any) => {
    setFormData((prev) => {
      const prevVal = prev[key]

      if (Array.isArray(prevVal)) {
      }

      return {
        ...prev,
        [key]: val,
      }
    })
  }

  useEffect(() => {
    const signals: (typeof INITIAL)['signals'] = []

    Object.entries(exportedSignals).forEach(([k, v]) => {
      if (v) signals.push(k.toUpperCase() as Signal)
    })

    handleFormChange('signals', signals)
  }, [exportedSignals])

  return {
    formData,
    handleFormChange,
    resetFormData,
    exportedSignals,
    setExportedSignals,
  }
}
