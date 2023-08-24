import styled from 'styled-components';

export const SourcesContainerWrapper = styled.div`
  height: 100vh;
  width: 100%;
  overflow: hidden;
`;

export const MenuWrapper = styled.div`
  padding: 0 32px;
`;

export const SourcesSectionWrapper = styled(MenuWrapper)`
  position: relative;
`;

export const ButtonWrapper = styled.div`
  position: absolute;
  display: flex;
  align-items: center;
  gap: 16px;
  right: 32px;
  top: 40px;
`;

export const ManageSourcePageContainer = styled.div`
  padding: 32px;
`;

export const BackButtonWrapper = styled.div`
  display: flex;
  width: fit-content;
  align-items: center;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

export const FieldWrapper = styled.div`
  height: 36px;
  width: 348px;
  margin-bottom: 64px;
`;

export const SaveSourceButtonWrapper = styled.div`
  margin-top: 48px;
  height: 36px;
  width: 362px;
`;
