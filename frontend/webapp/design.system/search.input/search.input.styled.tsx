import styled from "styled-components";

interface ActiveProps {
  active?: any;
}

export const SearchInputWrapper = styled.div<ActiveProps>`
  position: relative;
  display: flex;
  width: 340px;
  padding: 9px 13px;
  gap: 10px;
  border-radius: 8px;
  border: ${({ active, theme }) =>
    `1px solid ${active ? theme.colors.white : theme.colors.blue_grey}`};
  background: ${({ active, theme }) =>
    `${active ? theme.colors.dark : theme.colors.light_dark}`};
  &:hover {
    border: ${({ theme }) => `solid 1px ${theme.colors.white}`};
  }
`;

export const StyledSearchInput = styled.input<ActiveProps>`
  width: 85%;
  background: ${({ active, theme }) =>
    `${active ? theme.colors.dark : "transparent"}`};
  border: none;
  outline: none;
  color: ${({ active, theme }) =>
    `${active ? theme.colors.white : theme.text.grey}`};
  font-size: 14px;
  font-family: ${({ theme }) => theme.font_family.primary}, sans-serif;
  font-weight: 400;
  &:focus {
    color: ${({ theme }) => `solid 1px ${theme.colors.white}`};
  }
`;

export const LoaderWrapper = styled.div`
  position: absolute;
  right: 30px;
  top: 4px;
`;
