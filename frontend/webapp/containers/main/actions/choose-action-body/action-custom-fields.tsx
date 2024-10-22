import styled from 'styled-components'
import { InputList, KeyValueInputsList, Text } from '@/reuseable-components'
import type { ActionsType } from '@/types'

interface ActionCustomFieldsProps {
  actionType?: ActionsType
}

const FieldWrapper = styled.div`
  width: 100%;
  margin: 8px 0;
`

const FieldTitle = styled(Text)`
  margin-bottom: 12px;
`

const ActionCustomFields: React.FC<ActionCustomFieldsProps> = ({ actionType }) => {
  switch (actionType) {
    case 'AddClusterInfo':
      return (
        <FieldWrapper>
          <FieldTitle>Attributes to add</FieldTitle>
          <KeyValueInputsList required value={[]} onChange={(arr) => console.log(arr)} />
        </FieldWrapper>
      )

    case 'DeleteAttribute':
      return (
        <FieldWrapper>
          <FieldTitle>Attributes to delete</FieldTitle>
          <InputList required value={[]} onChange={(arr) => console.log(arr)} />
        </FieldWrapper>
      )

    case 'RenameAttribute':
      return (
        <FieldWrapper>
          <FieldTitle>Attributes to rename</FieldTitle>
          <KeyValueInputsList required value={[]} onChange={(arr) => console.log(arr)} />
        </FieldWrapper>
      )

    case 'PiiMasking':
      return (
        <FieldWrapper>
          <FieldTitle>Attributes to mask</FieldTitle>
          <InputList required value={[]} onChange={(arr) => console.log(arr)} />
        </FieldWrapper>
      )

    case 'ErrorSampler':
      return null

    case 'ProbabilisticSampler':
      return null

    case 'LatencySampler':
      return null

    default:
      return null
  }
}

export default ActionCustomFields
