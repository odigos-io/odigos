import styled from "styled-components";

interface DropdownWrapperProps {
  isHover?: boolean;
}

export const DropdownWrapper = styled.div<DropdownWrapperProps>`
  position: relative;
  z-index: 9999;
  width: 260px;
  padding: 11px 13px;
  border-radius: 8px;
  border: ${({ isHover, theme }) =>
    `1px solid  ${isHover ? theme.colors.white : theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.dark};

  .dropdown-arrow {
    transform: rotate(0deg);
    transition: all 0.2s ease-in-out;
  }

  .dropdown-arrow.open {
    transform: rotate(-90deg);
  }
`;

export const DropdownHeader = styled.div`
  cursor: pointer;
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: ${({ theme }) => theme.text.white};
  font-size: 14px;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-weight: 400;
`;

export const DropdownBody = styled.div`
  position: relative;
  z-index: 9999;
  display: flex;
  width: 278px;
  padding: 11px 4px;
  flex-direction: column;
  border-radius: 8px;
  border: ${({ theme }) => `1px solid ${theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.dark};
  margin-top: 5px;
`;

export const DropdownListWrapper = styled.div`
  position: relative;
  z-index: 9999;
  width: 278px;
  max-height: 270px;
  overflow-y: scroll;
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
