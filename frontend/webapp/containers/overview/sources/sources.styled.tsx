import styled from "styled-components";

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
