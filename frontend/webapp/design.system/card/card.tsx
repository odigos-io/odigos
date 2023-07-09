import React from "react";
import { CardContainer } from "./card.styled";
interface CardProps {
  children: React.ReactNode;
  focus?: any;
}
export function KeyvalCard({ children, focus }: CardProps) {
  return <CardContainer focus={focus}>{children}</CardContainer>;
}
