import React from 'react';
import styled from 'styled-components';
import { KeyvalImage } from '@/design.system';
import { ConnectIcon } from '@keyval-dev/design-system';

const ConnectionsIconsWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const Divider = styled.div`
  width: 16px;
  height: 8.396px;
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
};

export function ConnectionsIcons({
  icon,
  imageStyle,
}: {
  icon: string;
  imageStyle?: React.CSSProperties;
}) {
  return icon ? (
    <ConnectionsIconsWrapper>
      <ConnectIcon size={48} />
      <Divider />
      <KeyvalImage
        src={icon}
        width={40}
        height={40}
        style={{ ...IMAGE_STYLE, ...imageStyle }}
      />
    </ConnectionsIconsWrapper>
  ) : null;
}
