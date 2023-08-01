import React from "react";
import styled from "styled-components";
import { KeyvalText } from "@/design.system";
import { Back } from "@/assets/icons/overview";
import { SETUP } from "@/utils/constants";

export interface OverviewHeaderProps {
  title?: string;
  onBackClick?: any;
  isDisabled?: boolean;
}

const OverviewHeaderContainer = styled.div`
  display: flex;
  flex-direction: column;
  width: 100%;
  padding: 24px;
  border-bottom: 2px solid rgba(255, 255, 255, 0.08);
  background: ${({ theme }) => theme.colors.light_dark};
`;

const BackButtonWrapper = styled.div`
  display: flex;
  margin-bottom: 8px;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

export function OverviewHeader({ title, onBackClick }: OverviewHeaderProps) {
  return (
    <OverviewHeaderContainer>
      {onBackClick && (
        <BackButtonWrapper onClick={onBackClick}>
          <Back width={14} />
          <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
        </BackButtonWrapper>
      )}
      <KeyvalText size={32} weight={700}>
        {title}
      </KeyvalText>
    </OverviewHeaderContainer>
  );
}
