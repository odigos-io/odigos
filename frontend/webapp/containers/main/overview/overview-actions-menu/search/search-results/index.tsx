import React, { Fragment, useMemo, useState } from 'react';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { AbsoluteContainer } from '../../styled';
import styled, { useTheme } from 'styled-components';
import { buildSearchResults, type Category } from './builder';
import { Divider, SelectionButton, Text } from '@/reuseable-components';
import { getEntityIcon, getEntityItemId, getEntityLabel } from '@/utils';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useNodeDataFlowHandlers, useSourceCRUD } from '@/hooks';

interface Props {
  searchText: string;
  onClose: () => void;
}

const HorizontalScroll = styled.div`
  display: flex;
  align-items: center;
  padding: 12px;
  border-bottom: ${({ theme }) => `1px solid ${theme.colors.border}`};
  overflow-x: scroll;
`;

const VerticalScroll = styled.div`
  display: flex;
  flex-direction: column;
  padding: 12px;
  overflow-y: scroll;
`;

export const SearchResults = ({ searchText, onClose }: Props) => {
  const theme = useTheme();
  const { sources } = useSourceCRUD();
  const { actions } = useActionCRUD();
  const { destinations } = useDestinationCRUD();
  const { instrumentationRules } = useInstrumentationRuleCRUD();
  const { handleNodeClick } = useNodeDataFlowHandlers();

  const [selectedCategory, setSelectedCategory] = useState<Category>('all');

  const { categories, searchResults } = useMemo(
    () =>
      buildSearchResults({
        rules: instrumentationRules,
        sources,
        actions,
        destinations,
        searchText,
        selectedCategory,
      }),
    [instrumentationRules, sources, actions, destinations, searchText, selectedCategory],
  );

  return (
    <AbsoluteContainer>
      <HorizontalScroll style={{ borderBottom: `1px solid ${!searchResults.length ? 'transparent' : theme.colors.border}` }}>
        {categories.map(({ category, label, count }) => (
          <SelectionButton key={`category-select-${category}`} label={label} badgeLabel={count} isSelected={selectedCategory === category} onClick={() => setSelectedCategory(category as Category)} />
        ))}
      </HorizontalScroll>

      {searchResults.map(({ category, label, entities }, catIdx) => (
        <Fragment key={`category-list-${category}`}>
          <VerticalScroll style={{ maxHeight: selectedCategory !== 'all' ? '240px' : '140px' }}>
            <Text size={12} family='secondary' color={theme.text.darker_grey} style={{ marginLeft: '16px' }}>
              {label}
            </Text>

            {entities.map((item, entIdx) => (
              <SelectionButton
                key={`entity-${catIdx}-${entIdx}`}
                icon={getEntityIcon(category as OVERVIEW_ENTITY_TYPES)}
                label={getEntityLabel(item, category as OVERVIEW_ENTITY_TYPES, { extended: true })}
                onClick={() => {
                  const id = getEntityItemId(item);
                  // @ts-ignore
                  handleNodeClick(null, { data: { type: category, id } });
                  onClose();
                }}
                style={{ width: '100%', justifyContent: 'flex-start' }}
                color='transparent'
              />
            ))}
          </VerticalScroll>

          <Divider thickness={catIdx === searchResults.length - 1 ? 0 : 1} length='90%' margin='8px auto' />
        </Fragment>
      ))}
    </AbsoluteContainer>
  );
};
