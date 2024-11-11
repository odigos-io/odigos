import React, { useEffect, useMemo, useRef, useState } from 'react';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { DropdownOption } from '@/types';
import { useFilterStore } from '@/store/useFilterStore';
import { useNamespace, useOnClickOutside, useSourceCRUD } from '@/hooks';
import { Button, Dropdown, SelectionButton } from '@/reuseable-components';
import { MONITORS_OPTIONS } from '@/utils';

const RelativeContainer = styled.div`
  position: relative;
`;

const CardWrapper = styled.div`
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 10;
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  border: ${({ theme }) => `1px solid ${theme.colors.border}`};
  border-radius: 24px;
  width: 360px;
`;

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

interface FiltersState {
  namespace: DropdownOption | undefined;
  types: DropdownOption[];
  monitors: DropdownOption[];
}

const getFilterCount = (params: FiltersState) => {
  let count = 0;
  if (!!params.namespace) count++;
  count += params.types.length;
  count += params.monitors.length;
  return count;
};

const Filters = () => {
  const { namespace, setNamespace, types, setTypes, monitors, setMonitors } = useFilterStore();
  const { allNamespaces } = useNamespace();
  const { sources } = useSourceCRUD();

  const namespaceOptions = useMemo(() => {
    const options: DropdownOption[] = [];

    allNamespaces?.forEach(({ name: id }) => {
      if (!options.find((opt) => opt.id === id)) options.push({ id, value: id });
    });

    return options;
  }, [allNamespaces]);

  const typesOptions = useMemo(() => {
    const options: DropdownOption[] = [];

    sources.forEach(({ kind: id }) => {
      if (!options.find((opt) => opt.id === id)) options.push({ id, value: id });
    });

    return options;
  }, [sources]);

  const metricsOptions = useMemo(() => {
    const options: DropdownOption[] = [];

    MONITORS_OPTIONS.forEach(({ id, value }) => {
      if (!options.find((opt) => opt.id === id)) options.push({ id, value });
    });

    return options;
  }, []);

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
    // global
    setNamespace(filters.namespace);
    setTypes(filters.types);
    setMonitors(filters.monitors);
    // local
    setFilterCount(getFilterCount(filters));
    setFocused(false);
  };

  const onCancel = () => {
    setFocused(false);
  };

  const onReset = () => {
    // global
    setNamespace(undefined);
    setTypes([]);
    setMonitors([]);
    // local
    setFilters({ namespace: undefined, types: [], monitors: [] });
    setFilterCount(0);
  };

  const ref = useRef<HTMLDivElement>(null);
  useOnClickOutside(ref, onCancel);

  return (
    <RelativeContainer ref={ref}>
      <SelectionButton label='Filters' icon='/icons/common/filter.svg' badgeLabel={filterCount} badgeFilled withBorder color='transparent' onClick={toggleFocused} />

      {focused && (
        <CardWrapper>
          <Pad>
            <Dropdown
              title='Namespace'
              placeholder='Select namespace'
              options={namespaceOptions}
              value={filters['namespace']}
              onSelect={(val) => setFilters({ namespace: val, types: [], monitors: [] })}
              onDeselect={() => setFilters((prev) => ({ ...prev, namespace: undefined }))}
              required
              showSearch={false}
            />
            <Dropdown
              title='Type'
              placeholder='All'
              options={typesOptions}
              value={filters['types']}
              onSelect={(val) => setFilters((prev) => ({ ...prev, types: [...prev.types, val] }))}
              onDeselect={(val) => setFilters((prev) => ({ ...prev, types: prev.types.filter((opt) => opt.id !== val.id) }))}
              isMulti
              required
              showSearch={false}
            />
            <Dropdown
              title='Monitors'
              placeholder='All'
              options={metricsOptions}
              value={filters['monitors']}
              onSelect={(val) => setFilters((prev) => ({ ...prev, monitors: [...prev.monitors, val] }))}
              onDeselect={(val) => setFilters((prev) => ({ ...prev, monitors: prev.monitors.filter((opt) => opt.id !== val.id) }))}
              isMulti
              required
              showSearch={false}
            />
          </Pad>

          <Actions>
            <Button variant='primary' onClick={onApply} style={{ fontSize: 14 }}>
              Apply
            </Button>
            <Button variant='secondary' onClick={onCancel} style={{ fontSize: 14 }}>
              Cancel
            </Button>
            <Button variant='tertiary' onClick={onReset} style={{ fontSize: 14, color: theme.text.error, marginLeft: '100px' }}>
              Reset
            </Button>
          </Actions>
        </CardWrapper>
      )}
    </RelativeContainer>
  );
};

export { Filters };
