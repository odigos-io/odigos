import React, { useMemo, useRef, useState } from 'react';
import styled from 'styled-components';
import { DropdownOption } from '@/types';
import { useNamespace, useOnClickOutside } from '@/hooks';
import { Button, Dropdown, SelectionButton } from '@/reuseable-components';
import theme from '@/styles/theme';

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

const Filters = () => {
  const [namespace, setNamespace] = useState<DropdownOption | undefined>(undefined);
  const [types, setTypes] = useState<DropdownOption[]>([]);
  const [metrics, setMetrics] = useState<DropdownOption[]>([]);

  const [focused, setFocused] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const toggleFocused = () => setFocused((prev) => !prev);
  useOnClickOutside(ref, () => setFocused(false));

  const { allNamespaces } = useNamespace();
  const namespaceOptions = useMemo(() => allNamespaces?.map((ns) => ({ id: ns.name, value: ns.name })) || [], [allNamespaces]);

  const onApply = () => {
    alert('TODO !');
  };

  const onCancel = () => {
    onReset();
    setFocused(false);
  };

  const onReset = () => {
    setNamespace(undefined);
    setTypes([]);
    setMetrics([]);
  };

  return (
    <RelativeContainer ref={ref}>
      <SelectionButton label='Filters' icon='/icons/common/filter.svg' badgeLabel={0} badgeFilled withBorder color='transparent' onClick={toggleFocused} />

      {focused && (
        <CardWrapper>
          <Pad>
            <Dropdown title='Namespace' placeholder='Select namespace' options={namespaceOptions} value={namespace} onSelect={(val) => setNamespace(val)} required />

            {/* TODO: make these as multi-select dropwdowns (with internal checkboxes) */}
            <Dropdown title='Type' placeholder='All' options={[]} value={types[0]} onSelect={(val) => setTypes((prev) => prev)} required />
            <Dropdown title='Metric' placeholder='All' options={[]} value={metrics[0]} onSelect={(val) => setMetrics((prev) => prev)} required />
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
