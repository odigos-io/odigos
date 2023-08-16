import styled from 'styled-components';

export const SetupHeaderWrapper = styled.header`
  display: inline-flex;
  padding: 2vh 0px;
  align-items: center;
  width: 100%;
  max-width: 1288px;
  justify-content: space-between;
  border-radius: 24px;
  border: ${({ theme }) => `1px solid ${theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.light_dark};
`;

export const HeaderTitleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 24px;
  margin-left: 40px;
`;

export const HeaderButtonWrapper = styled.div`
  display: flex;
  gap: 16px;
  align-items: center;
  margin-right: 40px;
`;

export const TotalSelectedWrapper = styled.div`
  display: flex;
  gap: 6px;
`;
