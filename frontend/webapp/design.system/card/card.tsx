import React from "react";
import { Card } from "@keyval-org/design-system";
interface CardProps {
  children: any;
  focus?: any;
}
export function KeyvalCard(props: CardProps) {
  return <Card>{props.children}</Card>;
}
