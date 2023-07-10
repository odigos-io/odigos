import React from "react";
import { KeyvalText } from "../text/text";

interface KeyvalLinkProps {
  value: string;
  onClick?: () => void;
}

export function KeyvalLink({ value, onClick }: KeyvalLinkProps) {
  return (
    <KeyvalText color="#0EE6F3" onClick={onClick}>
      {value}
    </KeyvalText>
  );
}
