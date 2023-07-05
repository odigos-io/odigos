import React from "react";
import { CardContainer } from "./card.styled";
interface CardProps {
  children: React.ReactNode;
  focus?: boolean;
}
export function KeyvalCard({ children, focus = false }: CardProps) {
  return <CardContainer focus={focus}>{children}</CardContainer>;
}
