import React from 'react';
import styled from 'styled-components';
import { Input } from '@/reuseable-components';

interface Form {
  reportedName: string;
}

interface Props {
  formData: Form;
  handleFormChange: (key: keyof Form, val: any) => void;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 4px;
`;

export const UpdateSourceBody: React.FC<Props> = ({ formData, handleFormChange }) => {
  return (
    <Container>
      <Input
        title='Source name'
        tooltip='This overrides the default service name that runs in your cluster.'
        placeholder='Use a name that overrides the source name'
        value={formData.reportedName}
        onChange={({ target: { value } }) => handleFormChange('reportedName', value)}
      />
    </Container>
  );
};
