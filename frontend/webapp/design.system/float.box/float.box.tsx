import React from "react";
import { FloatBox } from "@keyval-org/design-system";

type FloatBoxProps = {
  style?: object;
  children: any;
};

export function FloatBoxComponent({ children, style = {} }: FloatBoxProps) {
  return <FloatBox style={{ ...style }}>{children}</FloatBox>;
}
