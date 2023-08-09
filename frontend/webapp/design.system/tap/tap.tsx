import React from "react";
import { Tap } from "@keyval-org/design-system";

interface TapProps {
  icons: object;
  title?: string;
  tapped?: any;
  onClick?: any;
  children?: JSX.Element;
  style?: React.CSSProperties;
}

export function KeyvalTap(props: TapProps) {
  return <Tap {...props} />;
}
