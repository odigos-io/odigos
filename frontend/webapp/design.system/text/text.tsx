import React from "react";
import { TextWrapper } from "./text.styled";

type TextProps = {
  type?: string | any;
  value?: string;
  style?: object;
  children?: string | any;
  weight?: string | number;
  color?: string;
  size?: number;
  onClick?: () => void;
};

export function KeyvalText({
  children,
  color,
  style,
  weight,
  size,
  onClick,
}: TextProps) {
  return (
    <TextWrapper
      onClick={onClick}
      style={{
        fontWeight: weight,
        color,
        fontSize: size,
        cursor: onClick ? "pointer" : "auto",
        ...style,
      }}
    >
      {children}
    </TextWrapper>
  );
}
