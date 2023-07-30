import React from "react";
import {
  InputsWrapper,
  SourcesMenuWrapper,
} from "./sources.action.menu.styled";
import { KeyvalButton, KeyvalText } from "@/design.system";
import { Plus } from "@/assets/icons/overview";
import { OVERVIEW } from "@/utils/constants";
import theme from "@/styles/palette";

const BUTTON_STYLES = { gap: 10, height: 36 };

export function SourcesActionMenu({ onAddClick }) {
  return (
    <SourcesMenuWrapper>
      <InputsWrapper></InputsWrapper>
      <KeyvalButton onClick={onAddClick} style={BUTTON_STYLES}>
        <Plus />
        <KeyvalText size={16} weight={700} color={theme.text.dark_button}>
          {OVERVIEW.ADD_NEW_SOURCE}
        </KeyvalText>
      </KeyvalButton>
    </SourcesMenuWrapper>
  );
}
