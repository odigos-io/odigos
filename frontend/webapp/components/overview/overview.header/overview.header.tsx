import React from "react";
import styled from "styled-components";
import { KeyvalText } from "@/design.system";

export interface OverviewHeaderProps {
  title?: string;
}

const OverviewHeaderContainer = styled.div`
  position: fixed;
  display: flex;
  width: 100%;
  height: 88px;
  align-items: center;
  padding: 0 24px;
  border-bottom: 2px solid rgba(255, 255, 255, 0.08);
  background: ${({ theme }) => theme.colors.light_dark};
`;

export function OverviewHeader({ title }: OverviewHeaderProps) {
  return (
    <OverviewHeaderContainer>
      <KeyvalText size={32} weight={700}>
        {title}
      </KeyvalText>
    </OverviewHeaderContainer>
  );
}
