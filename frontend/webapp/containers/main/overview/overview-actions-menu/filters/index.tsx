import React, { useEffect, useRef, useState } from 'react';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { useOnClickOutside } from '@/hooks';
import { FiltersState, useFilterStore } from '@/store/useFilterStore';
import { AbsoluteContainer, RelativeContainer } from '../styled';
import { Button, SelectionButton } from '@/reuseable-components';
import { MonitorDropdown, NamespaceDropdown, TypeDropdown } from '@/components';

const Pad = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
`;

const Actions = styled.div`
  display: flex;
  align-items: center;
  padding: 12px;
  border-top: ${({ theme }) => `1px solid ${theme.colors.border}`};
`;

const getFilterCount = (params: FiltersState) => {
  let count = 0;
  if (!!params.namespace) count++;
  count += params.types.length;
  count += params.monitors.length;
  return count;
};

const Filters = () => {
  const { namespace, types, monitors, setAll, clearAll } = useFilterStore();

  const [filters, setFilters] = useState<FiltersState>({ namespace, types, monitors });
  const [filterCount, setFilterCount] = useState(getFilterCount(filters));
  const [focused, setFocused] = useState(false);
  const toggleFocused = () => setFocused((prev) => !prev);

  useEffect(() => {
    if (!focused) {
      const payload = { namespace, types, monitors };
      setFilters(payload);
      setFilterCount(getFilterCount(payload));
    }
  }, [focused, namespace, types, monitors]);

  const onApply = () => {
    setAll(filters);
    setFilterCount(getFilterCount(filters));
    setFocused(false);
  };

  const onCancel = () => {
    setFocused(false);
  };

  const onReset = () => {
    clearAll();
    setFilters({ namespace: undefined, types: [], monitors: [] });
    setFilterCount(0);
  };

  const ref = useRef<HTMLDivElement>(null);
  useOnClickOutside(ref, onCancel);

  return (
    <RelativeContainer ref={ref}>
      <SelectionButton label='Filters' icon='/icons/common/filter.svg' badgeLabel={filterCount} badgeFilled withBorder color='transparent' onClick={toggleFocused} />

      {focused && (
        <AbsoluteContainer>
          <Pad>
            <NamespaceDropdown
              value={filters['namespace']}
              onSelect={(val) => setFilters({ namespace: val, types: [], monitors: [] })}
              onDeselect={(val) => setFilters((prev) => ({ ...prev, namespace: undefined }))}
              required
            />
            <TypeDropdown
              value={filters['types']}
              onSelect={(val) => setFilters((prev) => ({ ...prev, types: [...prev.types, val] }))}
              onDeselect={(val) => setFilters((prev) => ({ ...prev, types: prev.types.filter((opt) => opt.id !== val.id) }))}
              required
              isMulti
            />
            <MonitorDropdown
              value={filters['monitors']}
              onSelect={(val) => setFilters((prev) => ({ ...prev, monitors: [...prev.monitors, val] }))}
              onDeselect={(val) => setFilters((prev) => ({ ...prev, monitors: prev.monitors.filter((opt) => opt.id !== val.id) }))}
              required
              isMulti
            />
          </Pad>

          <Actions>
            <Button variant='primary' onClick={onApply} style={{ fontSize: 14 }}>
              Apply
            </Button>
            <Button variant='secondary' onClick={onCancel} style={{ fontSize: 14 }}>
              Cancel
            </Button>
            <Button variant='tertiary' onClick={onReset} style={{ fontSize: 14, color: theme.text.error, marginLeft: '160px' }}>
              Reset
            </Button>
          </Actions>
        </AbsoluteContainer>
      )}
    </RelativeContainer>
  );
};

export { Filters };
