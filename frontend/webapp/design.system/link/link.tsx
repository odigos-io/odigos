import React from "react";
import { KeyvalText } from "../text/text";
import { styled } from "styled-components";

interface KeyvalLinkProps {
  value: string;
  onClick?: () => void;
}

const LinkContainer = styled.div`
  cursor: pointer;
  .p {
    cursor: pointer !important;
  }
`;

export function KeyvalLink({ value, onClick }: KeyvalLinkProps) {
  return (
    <LinkContainer onClick={onClick}>
      <KeyvalText color="#0EE6F3">{value}</KeyvalText>
    </LinkContainer>
  );
}
