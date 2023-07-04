import React from "react";
import { TextWrapper } from "./text.styled";

type TextProps = {
  type?: string | any;
  value?: string;
  style?: object;
  children?: string | any;
  weight?: string | number;
};

export function KeyvalText({ children, type, style, weight }: TextProps) {
  return (
    <TextWrapper style={{ fontWeight: weight, ...style }}>
      {children}
    </TextWrapper>
  );
}
