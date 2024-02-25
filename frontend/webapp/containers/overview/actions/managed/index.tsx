import React from 'react';
import { Plus } from '@/assets';
import { OVERVIEW } from '@/utils';
import theme from '@/styles/palette';
import { KeyvalButton, KeyvalText } from '@/design.system';

export function ManagedActionsContainer() {
  return (
    <>
      <div>
        <KeyvalButton>
          <Plus />
          <KeyvalText size={16} weight={700} color={theme.text.dark_button}>
            {OVERVIEW.ADD_NEW_ACTION}
          </KeyvalText>
        </KeyvalButton>
      </div>
      <div></div>
    </>
  );
}
