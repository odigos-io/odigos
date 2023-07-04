import React from "react";
import { CardContainer } from "./card.styled";
interface CardProps {
  children: React.ReactNode;
}
export function KeyvalCard({ children }: CardProps) {
  return <CardContainer>{children}</CardContainer>;
}
