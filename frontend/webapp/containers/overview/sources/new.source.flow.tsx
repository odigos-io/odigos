import React from 'react';
import theme from '@/styles/palette';
import { KeyvalButton, KeyvalText } from '@/design.system';
import { NOTIFICATION, OVERVIEW, SETUP } from '@/utils/constants';
import { useNotification, useSectionData, useSources } from '@/hooks';
import { SourcesSectionWrapper, ButtonWrapper } from './sources.styled';
import { SourcesSection } from '@/containers/setup/sources/sources.section';

export function NewSourcesList({ onSuccess }) {
  const { sectionData, setSectionData, totalSelected } = useSectionData({});
  const { upsertSources } = useSources();

  const { show, Notification } = useNotification();

  function onError({ response }) {
    const message = response?.data?.message || SETUP.ERROR;
    show({
      type: NOTIFICATION.ERROR,
      message,
    });
  }

  return (
    <SourcesSectionWrapper>
      <ButtonWrapper>
        <KeyvalText>{`${totalSelected} ${SETUP.SELECTED}`}</KeyvalText>
        <KeyvalButton
          disabled={totalSelected === 0}
          onClick={() => upsertSources({ sectionData, onSuccess, onError })}
          style={{ width: 110 }}
        >
          <KeyvalText weight={600} color={theme.text.dark_button}>
            {OVERVIEW.CONNECT}
          </KeyvalText>
        </KeyvalButton>
      </ButtonWrapper>

      <SourcesSection
        sectionData={sectionData}
        setSectionData={setSectionData}
      />
      <Notification />
    </SourcesSectionWrapper>
  );
}
