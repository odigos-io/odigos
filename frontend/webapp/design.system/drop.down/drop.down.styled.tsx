import styled from "styled-components";

export const DropdownWrapper = styled.div`
  display: flex;
  width: 260px;
  padding: 11px 13px;
  flex-direction: column;
  gap: 10px;
  border-radius: 8px;
  border: 1px solid var(--dark-mode-dark-4, #374a5b);
  background: var(--dark-mode-dark-1, #0a1824);

  .dropdown-body {
    width: 100%;
    border-top: 1px solid #e5e8ec;
    display: none;
  }

  .dropdown-body.open {
    display: block;
  }

  .dropdown-item {
    padding: 10px 0;
  }

  .dropdown-item:hover {
    cursor: pointer;
  }

  .dropdown-item-dot {
    opacity: 0;
    color: #91a5be;
    transition: all 0.2s ease-in-out;
  }

  .dropdown-item-dot.selected {
    opacity: 1;
  }

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
