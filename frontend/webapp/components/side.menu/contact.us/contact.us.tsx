import React from 'react';
import { Slack, SlackGrey } from '@/assets/icons/social';
import { KeyvalText } from '@/design.system';
import { SLACK_INVITE_LINK } from '@/utils/constants/urls';
import { styled } from 'styled-components';
import { ACTION } from '@/utils/constants';

const ContactUsContainer = styled.div`
  display: flex;
  padding: 0px 16px;
  height: 48px;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  border-radius: 10px;
  p {
    color: #8b92a6;
  }
  .icon-lock {
    display: none;
  }
  &:hover {
    p {
      color: ${({ theme }) => theme.colors.white};
    }
    .icon-unlock {
      display: none;
    }

    .icon-lock {
      display: block;
    }
  }
`;

export default function ContactUsButton({ expand }) {
  function handleContactUsClick() {
    window.open(SLACK_INVITE_LINK, '_blank');
  }

  return (
    <ContactUsContainer onClick={handleContactUsClick}>
      <Slack width={24} height={24} className="icon-lock" />
      <SlackGrey width={24} height={24} className="icon-unlock" />
      {expand && <KeyvalText size={14}>{ACTION.CONTACT_US}</KeyvalText>}
    </ContactUsContainer>
  );
}
