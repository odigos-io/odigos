import React from 'react';
import Image from 'next/image';
import { SLACK_LINK } from '@/utils';
import { IconButton } from '@/reuseable-components';

export const SlackInvite = () => {
  const handleClick = () => window.open(SLACK_LINK, '_blank', 'noopener noreferrer');

  return (
    <IconButton onClick={handleClick} tooltip='Join our Slack community'>
      <Image src='/icons/social/slack.svg' alt='slack' width={16} height={16} />
    </IconButton>
  );
};
