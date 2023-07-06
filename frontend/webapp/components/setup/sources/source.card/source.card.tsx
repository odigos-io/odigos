import {
  KeyvalCard,
  KeyvalRadioButton,
  KeyvalTag,
  KeyvalText,
} from "@/design.system";
import React from "react";
import {
  ApplicationNameWrapper,
  RadioButtonWrapper,
  SourceCardWrapper,
} from "./source.card.styled";
import Logo from "assets/logos/code-sandbox-logo.svg";

const KIND_COLORS = {
  deployment: "#203548",
};

export function SourceCard({ item }: any) {
  return (
    <KeyvalCard focus={item?.selected}>
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
            {item.name}
          </KeyvalText>
        </ApplicationNameWrapper>
        <KeyvalTag title={item.kind} color={KIND_COLORS[item.kind]} />
        <KeyvalText size={14} weight={400}>
          {`${item?.instances} Running instance`}
        </KeyvalText>
      </SourceCardWrapper>
    </KeyvalCard>
  );
}
