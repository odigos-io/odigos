import React, { Fragment } from 'react'
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
  const { actionName, setActionName, actionNotes, setActionNotes, exportedSignals, setExportedSignals } = useActionFormData()

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
        <Input placeholder='Use a name that describes the action' value={actionName} onChange={({ target: { value } }) => setActionName(value)} />
      </FieldWrapper>

      <ActionCustomFields actionType={action.type} />

      <FieldWrapper>
        <FieldTitle>Notes</FieldTitle>
        <TextArea value={actionNotes} onChange={({ target: { value } }) => setActionNotes(value)} />
      </FieldWrapper>
    </Fragment>
  )
}

export { ChooseActionBody }
