import React from 'react';
import { KeyvalImage, KeyvalTag, KeyvalText } from '@/design.system';
import {
  CardWrapper,
  SourceManageContentWrapper,
} from './sources.manage.styled';
import theme from '@/styles/palette';
import { KIND_COLORS } from '@/styles/global';
import { LANGUAGES_LOGOS } from '@/assets/images';
import { ManagedSource } from '@/types/sources';

const TEXT_STYLE: React.CSSProperties = {
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  width: 224,
  textAlign: 'center',
};
const LOGO_STYLE: React.CSSProperties = {
  padding: 4,
  backgroundColor: theme.colors.white,
};

interface SourceManagedCardProps {
  item: ManagedSource;
  onClick?: () => void;
}
const DEPLOYMENT = 'deployment';
export default function SourceManagedCard({
  item = {} as ManagedSource,
  onClick,
}: SourceManagedCardProps) {
  return (
    <CardWrapper onClick={onClick}>
      <KeyvalImage
        src={LANGUAGES_LOGOS[item?.languages?.[0].language || '']}
        width={56}
        height={56}
        style={LOGO_STYLE}
        alt="source-logo"
      />
      <SourceManageContentWrapper>
        <KeyvalText size={18} weight={700} style={TEXT_STYLE}>
          {item?.name}
        </KeyvalText>
        <KeyvalTag
          title={item?.kind || ''}
          color={KIND_COLORS[item?.kind?.toLowerCase() || DEPLOYMENT]}
        />
        <KeyvalText size={14} color={theme.text.light_grey} style={TEXT_STYLE}>
          {item?.namespace}
        </KeyvalText>
      </SourceManageContentWrapper>
    </CardWrapper>
  );
}
