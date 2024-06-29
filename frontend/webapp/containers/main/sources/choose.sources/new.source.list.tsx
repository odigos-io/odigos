import React from 'react';
import theme from '@/styles/palette';
import { OVERVIEW, SETUP } from '@/utils';
import { useSectionData, useSources } from '@/hooks';
import { KeyvalButton, KeyvalText } from '@/design.system';
import { SourcesSectionWrapper, ButtonWrapper } from './styled';
import { SourcesSection } from '@/containers/setup/sources/sources.section';

export function NewSourcesList({ onSuccess }) {
  const { sectionData, setSectionData, totalSelected } = useSectionData({});
  const { upsertSources } = useSources();

  return (
    <>
      <ButtonWrapper>
        <KeyvalText>{`${totalSelected} ${SETUP.SELECTED}`}</KeyvalText>
        <KeyvalButton
          disabled={totalSelected === 0}
          onClick={() =>
            upsertSources({ sectionData, onSuccess, onError: null })
          }
          style={{ width: 110 }}
        >
          <KeyvalText weight={600} color={theme.text.dark_button}>
            {OVERVIEW.CONNECT}
          </KeyvalText>
        </KeyvalButton>
      </ButtonWrapper>
      <SourcesSectionWrapper>
        <SourcesSection
          sectionData={sectionData}
          setSectionData={setSectionData}
        />
      </SourcesSectionWrapper>
    </>
  );
}
