import {
  KeyvalCard,
  KeyvalRadioButton,
  KeyvalTag,
  KeyvalText,
} from "design.system";
import React from "react";
import {
  ApplicationNameWrapper,
  RadioButtonWrapper,
  SourceCardWrapper,
} from "./source.card.styled";
import Logo from "assets/logos/code-sandbox-logo.svg";

export function SourceCard() {
  return (
    <KeyvalCard>
      <RadioButtonWrapper>
        <KeyvalRadioButton />
      </RadioButtonWrapper>
      <SourceCardWrapper>
        <Logo />
        <ApplicationNameWrapper>
          <KeyvalText
            size={20}
            weight={700}
            style={{
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
              overflow: "hidden",
            }}
          >
            {"local - path - DaemonSet sdfsd sdfds"}
          </KeyvalText>
        </ApplicationNameWrapper>
        <KeyvalTag title="DaemonSet" />
        <KeyvalText size={14} weight={400}>
          {"1 Running instance"}
        </KeyvalText>
      </SourceCardWrapper>
    </KeyvalCard>
  );
}
