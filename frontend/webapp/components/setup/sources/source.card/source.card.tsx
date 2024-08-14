import {
  KeyvalCard,
  KeyvalRadioButton,
  KeyvalTag,
  KeyvalText,
} from '@/design.system';
import React from 'react';
import {
  ApplicationNameWrapper,
  RadioButtonWrapper,
  SourceCardWrapper,
  SourceTextWrapper,
} from './source.card.styled';
import Logo from 'assets/logos/code-sandbox-logo.svg';
import { SETUP } from '@/utils/constants';
import { KIND_COLORS } from '@/styles/global';

const TEXT_STYLE = {
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
};

export function SourceCard({ item, onClick, focus }: any) {
  return (
    <KeyvalCard focus={focus}>
      <RadioButtonWrapper>
        <KeyvalRadioButton onChange={onClick} value={focus} />
      </RadioButtonWrapper>
      <SourceCardWrapper onClick={onClick} data-cy={'choose-source-' + item.name}>
        <Logo width={'6vh'} height={'6vh'} />
        <SourceTextWrapper>
          <ApplicationNameWrapper>
            <KeyvalText size={18} weight={700} style={TEXT_STYLE}>
              {item.name}
            </KeyvalText>
          </ApplicationNameWrapper>
          <KeyvalTag
            title={item.kind}
            color={KIND_COLORS[item.kind.toLowerCase()]}
          />
          <KeyvalText size={14} weight={400}>
            {`${item?.instances} ${SETUP.RUNNING_INSTANCES}`}
          </KeyvalText>
        </SourceTextWrapper>
      </SourceCardWrapper>
    </KeyvalCard>
  );
}
