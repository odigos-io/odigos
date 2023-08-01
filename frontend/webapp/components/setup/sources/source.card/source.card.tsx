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
import { SETUP } from "@/utils/constants";
import { KIND_COLORS } from "@/styles/global";

const TEXT_STYLE = {
  overflowWrap: "break-word",
};

export function SourceCard({ item, onClick, focus }: any) {
  return (
    <KeyvalCard focus={focus}>
      <RadioButtonWrapper>
        <KeyvalRadioButton onChange={onClick} value={focus} />
      </RadioButtonWrapper>
      <SourceCardWrapper onClick={onClick}>
        <Logo />
        <ApplicationNameWrapper>
          <KeyvalText
            size={item?.name?.length > 20 ? 16 : 20}
            weight={700}
            style={TEXT_STYLE}
          >
            {item?.name}
          </KeyvalText>
        </ApplicationNameWrapper>
        <KeyvalTag
          title={item.kind}
          color={KIND_COLORS[item.kind.toLowerCase()]}
        />
        <KeyvalText size={14} weight={400}>
          {`${item?.instances} ${SETUP.RUNNING_INSTANCES}`}
        </KeyvalText>
      </SourceCardWrapper>
    </KeyvalCard>
  );
}
