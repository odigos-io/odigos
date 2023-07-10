import React from "react";
import { CardContainer } from "./card.styled";
interface CardProps {
  children: React.ReactNode;
  focus?: any;
}
export function KeyvalCard({ children, focus = false }: CardProps) {
  return <CardContainer active={focus || undefined}>{children}</CardContainer>;
}
