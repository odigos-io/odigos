import React, { Fragment, useEffect, useMemo } from 'react'
import styled from 'styled-components'
import { CheckboxList, DocsButton, Input, Text, TextArea } from '@/reuseable-components'
import { MONITORING_OPTIONS } from '@/components/setup/destination/utils'
import { useActionFormData } from '@/hooks/actions/useActionFormData'
import ActionCustomFields from './action-custom-fields'
import { type ActionOption } from '../choose-action-modal/action-options'

const Description = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  line-height: 150%;
  display: flex;
`

const FieldWrapper = styled.div`
  width: 100%;
  margin: 8px 0;
`

const FieldTitle = styled(Text)`
  margin-bottom: 12px;
`

interface ChooseActionContentProps {
  action: ActionOption
}

const ChooseActionBody: React.FC<ChooseActionContentProps> = ({ action }) => {
  const { formData, handleFormChange, resetFormData, exportedSignals, setExportedSignals } = useActionFormData()

  useEffect(() => {
    return () => {
      resetFormData()
    }
  }, [])

  return (
    <Fragment>
      <Description>
        {action.docsDescription}
        <DocsButton endpoint={action.docsEndpoint} />
      </Description>

      <FieldWrapper>
        <FieldTitle>Monitoring</FieldTitle>
        <CheckboxList
          monitors={MONITORING_OPTIONS}
          exportedSignals={exportedSignals}
          handleSignalChange={(id, val) => {
            const found = MONITORING_OPTIONS.find((item) => item.id === id)
            if (found) setExportedSignals((prev) => ({ ...prev, [found.type]: val }))
          }}
        />
      </FieldWrapper>

      <FieldWrapper>
        <FieldTitle>Action name</FieldTitle>
        <Input
          placeholder='Use a name that describes the action'
          value={formData.name}
          onChange={({ target: { value } }) => handleFormChange('name', value)}
        />
      </FieldWrapper>

      <ActionCustomFields
        actionType={action.type}
        value={formData.details ? JSON.parse(formData.details) : undefined}
        setValue={(val) => handleFormChange('details', JSON.stringify(val))}
      />

      <FieldWrapper>
        <FieldTitle>Notes</FieldTitle>
        <TextArea value={formData.notes} onChange={({ target: { value } }) => handleFormChange('notes', value)} />
      </FieldWrapper>
    </Fragment>
  )
}

export { ChooseActionBody }
