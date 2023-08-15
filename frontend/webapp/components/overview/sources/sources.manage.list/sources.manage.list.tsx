import React from 'react';
import {
  ManagedListWrapper,
  EmptyListWrapper,
  ManagedContainer,
} from './sources.manage.styled';
import SourceManagedCard from './sources.manage.card';
import { ManagedSource } from '@/types/sources';
import { KeyvalButton, KeyvalText } from '@/design.system';
import { OVERVIEW, ROUTES } from '@/utils/constants';
import { useRouter } from 'next/navigation';
import { Plus } from '@/assets/icons/overview';
import theme from '@/styles/palette';
import { Empty } from '@/assets/images';

interface SourcesManagedListProps {
  data: ManagedSource[];
}
const BUTTON_STYLES = { gap: 10, width: 224, height: 40 };
export function SourcesManagedList({ data = [] }: SourcesManagedListProps) {
  const router = useRouter();
  function renderSources() {
    return data.map((source: ManagedSource) => (
      <SourceManagedCard
        key={source?.name}
        item={source}
        onClick={() =>
          router.push(
            `${ROUTES.MANAGE_SOURCE}?name=${source?.name}&kind=${source?.kind}&namespace=${source?.namespace}`
          )
        }
      />
    ));
  }

  return data.length === 0 ? (
    <EmptyListWrapper>
      <Empty />
      <br />
      <KeyvalText size={14}>{OVERVIEW.EMPTY_SOURCE}</KeyvalText>
      <br />
      <KeyvalButton onClick={() => {}} style={BUTTON_STYLES}>
        <Plus />
        <KeyvalText size={16} weight={700} color={theme.text.dark_button}>
          {OVERVIEW.ADD_NEW_SOURCE}
        </KeyvalText>
      </KeyvalButton>
    </EmptyListWrapper>
  ) : (
    <ManagedContainer>
      <KeyvalText>{`${data.length} ${OVERVIEW.MENU.SOURCES}`}</KeyvalText>
      <br />
      <ManagedListWrapper>{renderSources()}</ManagedListWrapper>
    </ManagedContainer>
  );
}
