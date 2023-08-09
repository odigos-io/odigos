import React from "react";
import { Card } from "@keyval-dev/design-system";
interface CardProps {
  children: any;
  focus?: any;
}
export function KeyvalCard(props: CardProps) {
  return <Card>{props.children}</Card>;
}
