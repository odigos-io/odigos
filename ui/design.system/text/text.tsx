import React from "react";
import { TextWrapper } from "./text.styled";

type TextProps = {
  type?: string | any;
  value?: string;
  style?: object;
  children?: string | any;
};

export function KeyvalText({ children, type, style }: TextProps) {
  return <TextWrapper>{children}</TextWrapper>;
}
