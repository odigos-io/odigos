import React from 'react';
import { SETUP } from '@/utils';
import { NoteIcon } from '@keyval-dev/design-system';
import { KeyvalText, KeyvalVideo } from '@/design.system';
import { QuickHelpHeader, QuickHelpVideoWrapper } from './quick.help.styled';
export function QuickHelp({ data }) {
  function renderVideoList() {
    return data?.map((video) => (
      <QuickHelpVideoWrapper key={video?.name}>
        <KeyvalVideo
          videoSrc={video?.src}
          title={video?.name}
          thumbnail={video?.thumbnail_url}
        />
      </QuickHelpVideoWrapper>
    ));
  }

  return (
    <div>
      <QuickHelpHeader>
        <NoteIcon />
        <KeyvalText size={18} weight={600}>
          {SETUP.QUICK_HELP}
        </KeyvalText>
      </QuickHelpHeader>
      {renderVideoList()}
    </div>
  );
}
