import React from 'react';
import { SlackLogo } from '@/assets';
import { SLACK_LINK } from '@/utils';
import { IconButton } from '@/reuseable-components';

export const SlackInvite = () => {
  const handleClick = () => window.open(SLACK_LINK, '_blank', 'noopener noreferrer');

  return (
    <IconButton onClick={handleClick} tooltip='Join our Slack community'>
      <SlackLogo />
    </IconButton>
  );
};
