import styled from 'styled-components';
import { InputList, KeyValueInputsList, Text } from '@/reuseable-components';
import { ActionsType } from '@/types';

interface ActionCustomFieldsProps {
  actionType?: ActionsType;
  value: any;
  setValue: (value: any) => void;
}

const FieldWrapper = styled.div`
  width: 100%;
  margin: 8px 0;
`;

const FieldTitle = styled(Text)`
  margin-bottom: 12px;
`;

const ActionCustomFields: React.FC<ActionCustomFieldsProps> = ({ actionType, value, setValue }) => {
  switch (actionType) {
    case ActionsType.ADD_CLUSTER_INFO:
      return (
        <FieldWrapper>
          <FieldTitle>Attributes to add</FieldTitle>
          <KeyValueInputsList required value={value} onChange={(arr) => setValue(arr)} />
        </FieldWrapper>
      );

    case ActionsType.DELETE_ATTRIBUTES:
      return (
        <FieldWrapper>
          <FieldTitle>Attributes to delete</FieldTitle>
          <InputList required value={value} onChange={(arr) => setValue(arr)} />
        </FieldWrapper>
      );

    case ActionsType.RENAME_ATTRIBUTES:
      return (
        <FieldWrapper>
          <FieldTitle>Attributes to rename</FieldTitle>
          <KeyValueInputsList required value={value} onChange={(arr) => setValue(arr)} />
        </FieldWrapper>
      );

    case ActionsType.PII_MASKING:
      return (
        <FieldWrapper>
          <FieldTitle>Attributes to mask</FieldTitle>
          <InputList required value={value} onChange={(arr) => setValue(arr)} />
        </FieldWrapper>
      );

    case ActionsType.ERROR_SAMPLER:
      return null;

    case ActionsType.PROBABILISTIC_SAMPLER:
      return null;

    case ActionsType.LATENCY_SAMPLER:
      return null;

    default:
      return null;
  }
};

export default ActionCustomFields;
