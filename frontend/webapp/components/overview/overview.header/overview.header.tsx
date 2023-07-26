import React, { CSSProperties } from "react";
import styled from "styled-components";
import { KeyvalButton, KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import theme from "@/styles/palette";
import RightArrow from "assets/icons/arrow-right.svg";
export interface OverviewHeaderProps {
  title?: string;
  onClick?: any;
  isDisabled?: boolean;
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

const BUTTON_STYLE: CSSProperties = {
  gap: 10,
  width: 120,
  position: "absolute",
  right: 310,
  height: 48,
  top: 20,
};

export function OverviewHeader({
  title,
  onClick,
  isDisabled,
}: OverviewHeaderProps) {
  return (
    <div>
      <OverviewHeaderContainer>
        <KeyvalText size={32} weight={700}>
          {title}
        </KeyvalText>
        {onClick && (
          <KeyvalButton
            disabled={isDisabled}
            onClick={onClick}
            style={BUTTON_STYLE}
          >
            <KeyvalText size={20} weight={600} color={theme.text.dark_button}>
              {SETUP.NEXT}
            </KeyvalText>
            <RightArrow />
          </KeyvalButton>
        )}
      </OverviewHeaderContainer>
    </div>
  );
}
