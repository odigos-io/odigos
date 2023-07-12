import { KeyvalText } from "@/design.system/text/text";
import React from "react";
import styled from "styled-components";

interface TapProps {
  icons: object;
  title?: string;
  tapped?: boolean;
  onClick?: () => void;
  children?: React.ReactNode;
}

interface TapWrapperProps {
  tapped?: boolean | undefined;
}

const TapWrapper = styled.div<TapWrapperProps>`
  display: flex;
  padding: 8px 14px;
  align-items: flex-start;
  gap: 10px;
  border-radius: 16px;
  border: ${({ theme, tapped }) =>
    `1px solid ${tapped ? "transparent" : theme.colors.dark_blue}`};
  background: ${({ theme, tapped }) =>
    tapped ? theme.colors.dark_blue : "transparent"};
`;

export function KeyvalTap({ title = "", tapped, children }: TapProps) {
  return (
    <TapWrapper tapped={tapped}>
      {children}
      <KeyvalText weight={400} size={14} color={tapped ? "#CCD0D2" : "#8B92A5"}>
        {title}
      </KeyvalText>
    </TapWrapper>
  );
}
