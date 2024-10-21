import styled from 'styled-components';
import { CheckboxList, Input } from '@/reuseable-components';
import { DynamicConnectDestinationFormFields } from '../dynamic-form-fields';

export const FormContainer = ({
  monitors,
  dynamicFields,
  exportedSignals,
  destinationName,
  handleDynamicFieldChange,
  handleSignalChange,
  setDestinationName,
}) => (
  <StyledFormContainer>
    <CheckboxList
      monitors={monitors}
      title="This connection will monitor:"
      exportedSignals={exportedSignals}
      handleSignalChange={handleSignalChange}
    />
    <Input
      title="Destination name"
      placeholder="Enter destination name"
      value={destinationName}
      onChange={(e) => setDestinationName(e.target.value)}
    />
    <DynamicConnectDestinationFormFields
      fields={dynamicFields}
      onChange={handleDynamicFieldChange}
    />
  </StyledFormContainer>
);

const StyledFormContainer = styled.div`
  display: flex;
  width: 100%;
  max-width: 500px;
  flex-direction: column;
  gap: 24px;
  height: 443px;
  overflow-y: auto;
  padding-right: 16px;
  box-sizing: border-box;
  overflow: overlay;
  max-height: calc(100vh - 410px);

  @media (height < 768px) {
    max-height: calc(100vh - 350px);
  }
`;
