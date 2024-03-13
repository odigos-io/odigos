import React from 'react';
import styled from 'styled-components';
import { KeyvalImage, KeyvalText } from '@/design.system';
import { ManagedSource } from '@/types/sources';
import { LANGUAGES_LOGOS } from '@/utils';

const ManageSourceHeaderWrapper = styled.div`
  display: flex;
  width: 100%;
  min-width: 686px;
  height: 104px;
  align-items: center;
  border-radius: 25px;
  margin: 24px 0;
  background: radial-gradient(
      78.09% 72.18% at 100% -0%,
      rgba(150, 242, 255, 0.4) 0%,
      rgba(150, 242, 255, 0) 61.91%
    ),
    linear-gradient(180deg, #2e4c55 0%, #303355 100%);
`;

const IMAGE_STYLE: React.CSSProperties = {
  backgroundColor: '#fff',
  padding: 4,
  marginRight: 16,
  marginLeft: 16,
};

export function ManageSourceHeader({ source }: { source: ManagedSource }) {
  const mainLanguage = source?.languages?.[0].language;
  const imageUrl = LANGUAGES_LOGOS[mainLanguage];
  return (
    <ManageSourceHeaderWrapper>
      <KeyvalImage src={imageUrl} style={IMAGE_STYLE} />
      <div style={{ flex: 1 }}>
        <KeyvalText size={24} weight={600} color="#fff">
          {source.name}
        </KeyvalText>
        <KeyvalText size={16} weight={400} color="#fff">
          {source.kind} in namespace &quot;{source.namespace}&quot;
        </KeyvalText>
      </div>
    </ManageSourceHeaderWrapper>
  );
}
