import { useState } from 'react'

export function useActionFormData() {
  const [actionName, setActionName] = useState('')
  const [actionNotes, setActionNotes] = useState('')
  const [exportedSignals, setExportedSignals] = useState({
    logs: false,
    metrics: false,
    traces: false,
  })

  return {
    actionName,
    setActionName,
    actionNotes,
    setActionNotes,
    exportedSignals,
    setExportedSignals,
    // resetFormData,
  }
}
