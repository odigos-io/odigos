import styled from "styled-components";

interface ActiveProps {
  active?: boolean;
}

export const SearchInputWrapper = styled.div<ActiveProps>`
  position: relative;
  display: flex;
  width: 340px;
  padding: 9px 13px;
  gap: 10px;
  border-radius: 8px;
  border: ${({ active }) => `1px solid ${active ? "#fff" : "#374a5b"}`};
  background: ${({ active }) => `${active ? "#07111A" : "#132330"}`};
  &:hover {
    border: 1px solid var(--dark-mode-white, #fff);
  }
`;

export const StyledSearchInput = styled.input<ActiveProps>`
  width: 85%;
  background: ${({ active }) => `${active ? "#07111A" : "#132330"}`};
  border: none;
  outline: none;
  color: ${({ active }) => `${active ? "#fff" : "#8b92a5"}`};
  font-size: 14px;
  font-family: Inter;
  font-weight: 400;
  &:focus {
    color: var(--dark-mode-grey-2, #fff);
  }
`;

export const LoaderWrapper = styled.div`
  position: absolute;
  right: 30px;
  top: 4%;
`;
