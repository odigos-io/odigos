import React from "react";
import { Text } from "@keyval-dev/design-system";

type TextProps = {
  type?: string | any;
  value?: string;
  style?: object;
  children?: string | any;
  weight?: string | number;
  color?: string;
  size?: number;
};

export function KeyvalText(props: TextProps) {
  return <Text {...props} />;
}
