import React from 'react';
import styled from 'styled-components';
import { KeyvalImage, KeyvalText } from '@/design.system';
import theme from '@/styles/palette';
import { LinkIcon } from '@keyval-dev/design-system';
import { DOCS_LINK } from '@/utils';

const ManageDestinationHeaderWrapper = styled.div`
  display: flex;
  height: 104px;
  align-items: center;
  border-radius: 25px;
  margin: 24px 0;
  background: radial-gradient(78.09% 72.18% at 100% -0%, rgba(150, 242, 255, 0.4) 0%, rgba(150, 242, 255, 0) 61.91%),
    linear-gradient(180deg, #2e4c55 0%, #303355 100%);
`;

const TextWrapper = styled.div``;

const IMAGE_STYLE: React.CSSProperties = {
  backgroundColor: '#fff',
  padding: 4,
  marginRight: 16,
  marginLeft: 16,
};

export function ManageDestinationHeader({ data: { image_url, display_name, type } }) {
  return (
    <ManageDestinationHeaderWrapper>
      <KeyvalImage src={image_url} style={IMAGE_STYLE} />
      <TextWrapper>
        <KeyvalText size={24} weight={700}>
          {display_name}
        </KeyvalText>
        <div style={{ cursor: 'pointer' }} onClick={() => window.open(`${DOCS_LINK}/backends/${type}`, '_blank')}>
          <KeyvalText style={{ display: 'flex', gap: 3 }}>
            find out more about {display_name} in <a style={{ color: theme.colors.torquiz_light }}>our docs</a>
            <LinkIcon style={{ marginTop: 2 }} />
          </KeyvalText>
        </div>
      </TextWrapper>
    </ManageDestinationHeaderWrapper>
  );
}
