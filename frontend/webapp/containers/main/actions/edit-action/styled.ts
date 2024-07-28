import theme from '@/styles/palette';
import styled from 'styled-components';

export const CreateActionWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 24px;
  box-sizing: border-box;
  max-height: 93%;
  width: 100%;
  overflow-y: auto;

  @media screen and (max-height: 450px) {
    max-height: 85%;
  }
`;

export const FormFieldsWrapper = styled.div<{ disabled: boolean }>`
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  opacity: ${({ disabled }) => (disabled ? 0.3 : 1)};
  pointer-events: ${({ disabled }) => (disabled ? 'none' : 'auto')};
`;

export const SwitchWrapper = styled.div<{
  disabled: boolean;
  isValid: boolean;
}>`
  p {
    color: ${({ disabled }) =>
      disabled ? theme.colors.orange_brown : theme.colors.success};
    font-weight: 600;
  }

  opacity: ${({ isValid }) => (!isValid ? 0.3 : 1)};
  pointer-events: ${({ isValid }) => (!isValid ? 'none' : 'auto')};
`;

export const KeyvalInputWrapper = styled.div`
  width: 362px;
`;

export const TextareaWrapper = styled.div`
  width: 375px;
`;

export const CreateButtonWrapper = styled.div`
  margin-top: 32px;
  width: 375px;
`;

export const DescriptionWrapper = styled.div`
  width: 80vw;
  max-width: 1050px;
  margin-bottom: 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
`;

export const LinkWrapper = styled.div`
  margin-left: 8px;
  width: 100px;
`;

export const LoaderWrapper = styled.div`
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
`;

export const HeaderText = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;
