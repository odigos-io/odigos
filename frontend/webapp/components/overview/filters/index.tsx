import React, { useMemo, useRef, useState } from 'react';
import styled from 'styled-components';
import { DropdownOption } from '@/types';
import { useNamespace, useOnClickOutside } from '@/hooks';
import { Dropdown, SelectionButton } from '@/reuseable-components';

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
  padding: 12px;
  gap: 12px;
  display: flex;
  flex-direction: column;
`;

const Filters = () => {
  const [focused, setFocused] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const toggleFocused = () => setFocused((prev) => !prev);
  useOnClickOutside(ref, () => setFocused(false));

  const { allNamespaces } = useNamespace();
  const namespaceOptions = useMemo(() => allNamespaces?.map((ns) => ({ id: ns.name, value: ns.name })) || [], [allNamespaces]);

  const [namespace, setNamespace] = useState<DropdownOption | undefined>(undefined);
  const [filters, setFilters] = useState<DropdownOption[]>([]);
  const [metrics, setMetrics] = useState<DropdownOption[]>([]);

  return (
    <RelativeContainer ref={ref}>
      <SelectionButton label='Filters' icon='/icons/common/filter.svg' badgeLabel={0} badgeFilled withBorder color='transparent' onClick={toggleFocused} />

      {focused && (
        <CardWrapper>
          <Pad>
            <Dropdown title='Namespace' placeholder='Select namespace' options={namespaceOptions} value={namespace} onSelect={(val) => setNamespace(val)} required />

            {/* TODO: make these as multi-select dropwdowns (with internal checkboxes) */}
            <Dropdown title='Type' placeholder='All' options={[]} value={filters[0]} onSelect={(val) => setFilters((prev) => prev)} required />
            <Dropdown title='Metric' placeholder='All' options={[]} value={metrics[0]} onSelect={(val) => setMetrics((prev) => prev)} required />
          </Pad>
        </CardWrapper>
      )}
    </RelativeContainer>
  );
};

export { Filters };
