import styled from "styled-components";

interface DropdownWrapperProps {
  selected?: any;
}

export const DropdownWrapper = styled.div<DropdownWrapperProps>`
  position: relative;
  z-index: 9999;
  width: 100%;
  padding: 11px 4px;
  border-radius: 8px;
  cursor: pointer;
  border: ${({ selected, theme }) =>
    `1px solid  ${selected ? theme.colors.white : theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.dark};

  .dropdown-arrow {
    transform: rotate(0deg);
    transition: all 0.2s ease-in-out;
  }

  .dropdown-arrow.open {
    transform: rotate(180deg);
  }
`;

export const DropdownHeader = styled.div`
  padding: 0 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: ${({ theme }) => theme.text.white};
  font-size: 14px;
  font-family: ${({ theme }) => theme.font_family.primary}, sans-serif;
  font-weight: 400;
`;

export const DropdownBody = styled.div`
  position: relative;
  z-index: 9999;
  display: flex;
  width: 100%;
  padding: 11px 4px;
  flex-direction: column;
  border-radius: 8px;
  border: ${({ theme }) => `1px solid ${theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.dark};
  margin-top: 5px;
`;

export const DropdownListWrapper = styled.div`
  position: relative;
  z-index: 100;
  width: 100%;
  max-height: 270px;
  overflow-y: scroll;
  scrollbar-width: none;
  :hover {
    background: ${({ theme }) => theme.colors.dark_blue};
  }
`;

export const DropdownItem = styled.div`
  display: flex;
  padding: 7px 12px;
  justify-content: space-between;
  align-items: center;
  border-radius: 8px;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;
