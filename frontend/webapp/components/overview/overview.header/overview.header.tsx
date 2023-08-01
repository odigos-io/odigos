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
  border-bottom: 2px solid rgba(255, 255, 255, 0.08);
  background: ${({ theme }) => theme.colors.light_dark};
`;

const BackButtonWrapper = styled.div`
  display: flex;
  margin: 24px;
  margin-bottom: 0;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

const TextWrapper = styled.div`
  margin-top: 24px;
  margin-left: 24px;
  margin-bottom: 24px;
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
      <TextWrapper>
        <KeyvalText size={32} weight={700}>
          {title}
        </KeyvalText>
      </TextWrapper>
    </OverviewHeaderContainer>
  );
}
