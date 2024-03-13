import styled from 'styled-components';

export const Container = styled.div`
  display: flex;
  height: 100%;
  padding: 24px;
  .action-yaml-column {
    display: none;
  }
  @media screen and (max-height: 700px) {
    height: 90%;
  }

  @media screen and (max-width: 1200px) {
    .action-yaml-row {
      display: none;
    }
    .action-yaml-column {
      display: block;
    }
    width: 100%;
  }
`;

export const HeaderText = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

export const CreateActionWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 24px;
  padding-top: 0;
  box-sizing: border-box;
  max-height: 90%;
  overflow-y: auto;

  @media screen and (max-height: 450px) {
    max-height: 85%;
  }

  @media screen and (max-width: 1200px) {
    width: 100%;
  }
`;

export const ActionYamlWrapper = styled(CreateActionWrapper)``;

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
  width: 100%;
  max-width: 40vw;
  min-width: 370px;
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
