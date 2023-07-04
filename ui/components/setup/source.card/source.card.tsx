import { KeyvalCard, KeyvalRadioButton, KeyvalText } from "design.system";
import React from "react";
import { RadioButtonWrapper, SourceCardWrapper } from "./source.card.styled";
import Logo from "assets/logos/code-sandbox-logo.svg";
export function SourceCard() {
  return (
    <KeyvalCard>
      <RadioButtonWrapper>
        <KeyvalRadioButton />
      </RadioButtonWrapper>
      <SourceCardWrapper>
        <Logo />
        <KeyvalText>{"local - path - provisioner"}</KeyvalText>
      </SourceCardWrapper>
    </KeyvalCard>
  );
}
