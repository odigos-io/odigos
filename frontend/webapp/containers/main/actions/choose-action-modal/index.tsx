import React, { useRef, useCallback, useState } from 'react'
import styled from 'styled-components'
import { AutocompleteInput, Modal, NavigationButtons, Text, Divider } from '@/reuseable-components'
import { ChooseActionBody } from '../choose-action-body'
import { ACTION_OPTIONS, type ActionOption } from './action-options'

const DefineActionContainer = styled.section`
  height: 640px;
  padding: 0px 220px;
  display: flex;
  flex-direction: column;
  overflow-y: scroll;
`

const WidthConstraint = styled.div`
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  align-items: flex-start;
  gap: 16px;
  max-width: 640px;
  margin: 32px 0 24px 0;
`

const SubTitle = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  line-height: 150%;
`

interface AddActionModalProps {
  isModalOpen: boolean
  handleCloseModal: () => void
}

interface ModalActionComponentProps {
  onNext: () => void
}

const ModalActionComponent: React.FC<ModalActionComponentProps> = React.memo(({ onNext }) => {
  const buttons = [
    {
      label: 'DONE',
      onClick: onNext,
      variant: 'primary' as const,
    },
  ]

  return <NavigationButtons buttons={buttons} />
})

export const AddActionModal: React.FC<AddActionModalProps> = ({ isModalOpen, handleCloseModal }) => {
  const submitRef = useRef<(() => void) | null>(null)
  const [selectedItem, setSelectedItem] = useState<ActionOption | null>(null)

  const handleNext = useCallback(() => {
    if (submitRef.current) {
      handleCloseModal()
    }
  }, [handleCloseModal])

  const handleClose = useCallback(() => {
    handleCloseModal()
    setSelectedItem(null)
  }, [handleCloseModal])

  return (
    <Modal isOpen={isModalOpen} actionComponent={<ModalActionComponent onNext={handleNext} />} header={{ title: 'Add Action' }} onClose={handleClose}>
      <DefineActionContainer>
        <WidthConstraint>
          <Text size={20}>{'Define Action'}</Text>
          <SubTitle>
            {
              'Actions are a way to modify the OpenTelemetry data recorded by Odigos sources before it is exported to your Odigos destinations. Choose an action type and provide necessary information.'
            }
          </SubTitle>
        </WidthConstraint>

        <AutocompleteInput options={ACTION_OPTIONS} onOptionSelect={(item) => setSelectedItem(item)} />

        {!!selectedItem?.type ? (
          <WidthConstraint>
            <Divider margin='16px 0' />
            <ChooseActionBody action={selectedItem} />
          </WidthConstraint>
        ) : null}
      </DefineActionContainer>
    </Modal>
  )
}
