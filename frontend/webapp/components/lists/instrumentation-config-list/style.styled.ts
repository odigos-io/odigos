import styled from 'styled-components';

export const InstrumentationConfigItemContainer = styled.div`
  border-radius: 8px;
`;

export const InstrumentationConfigItemHeader = styled.div`
  display: flex;
  margin: 12px 0;
  cursor: pointer;
  span {
    width: 18px !important;
    height: 18px !important;
  }
  .dropdown-arrow {
    transform: rotate(0deg);
    transition: all 0.2s ease-in-out;
  }

  .dropdown-arrow.open {
    transform: rotate(180deg);
  }
`;

export const InstrumentationConfigItemContent = styled.div<{ open: boolean }>`
  display: ${({ open }) => (open ? 'flex' : 'none')};
  flex-direction: column;
  gap: 12px;
  padding: 0px 12px 12px 34px;
  border-bottom: 1px solid #e5e7eb1a;
`;

export const StyledItemCountContainer = styled.div`
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 0 8px;
  border-radius: 16px;
  height: 32px;
`;

export const StyledLibraryOptionContainer = styled.div`
  width: fit-content;
  display: flex;
  gap: 8px;

  span {
    width: 18px !important;
    height: 18px !important;
  }
`;

export const InstrumentationConfigHeaderContent = styled.div`
  display: flex;
  margin-left: 8px;
  width: 100%;
  justify-content: space-between;
  align-items: center;
`;

export const HeaderItemWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

export const TextInformationWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
`;

export const LibCheckboxWrapper = styled.div<{ disabled: boolean }>`
  display: flex;
  align-items: center;
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
`;

export const LibNameWrapper = styled.div`
  width: 200px;
  display: flex;
  flex-direction: column;
  gap: 2px;
`;
