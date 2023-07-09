import styled from "styled-components";

interface DropdownWrapperProps {
  hover?: boolean;
}

export const DropdownWrapper = styled.div<DropdownWrapperProps>`
  position: relative;
  z-index: 9999;
  width: 260px;
  padding: 11px 13px;
  border-radius: 8px;
  border: ${({ hover }) => `1px solid  ${hover ? "#fff" : "#374a5b"}`};
  background: var(--dark-mode-dark-1, #0a1824);

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
  border: 1px solid #374a5b;
  background: var(--dark-mode-dark-1, #0a1824);
  margin-top: 5px;
`;

export const DropdownListWrapper = styled.div`
  position: relative;
  z-index: 9999;
  width: 278px;
  max-height: 270px;
  overflow-y: scroll;
  :hover {
    background: var(--dark-mode-dark-3, #203548);
  }
`;

export const DropdownItem = styled.div`
  display: flex;
  padding: 7px 12px;
  flex-direction: column;
  align-items: flex-start;
  border-radius: 8px;
  cursor: pointer;
`;
