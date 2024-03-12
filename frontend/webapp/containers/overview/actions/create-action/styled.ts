import styled from 'styled-components';

export const Container = styled.div`
  display: flex;
  height: 100%;
  padding: 24px;
`;

export const CreateActionWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 24px;
  box-sizing: border-box;
  max-height: 90%;
  overflow-y: auto;

  @media screen and (max-height: 450px) {
    max-height: 85%;
  }
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
  width: 50vw;
  max-width: 600px;
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
