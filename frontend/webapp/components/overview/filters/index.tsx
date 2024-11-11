import React, { useRef, useState } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { DropdownOption } from '@/types';
import { useOnClickOutside } from '@/hooks';
import { Badge, Button, Dropdown, Text } from '@/reuseable-components';

const RelativeContainer = styled.div`
  position: relative;
`;

const ButtonText = styled(Text)`
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: 14px;
  text-transform: none;
  margin: 0 6px;
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

const CardContent = styled.div`
  padding: 16px;
  gap: 12px;
  display: flex;
  flex-direction: column;
`;

const Filters = () => {
  const [focused, setFocused] = useState(false);
  const toggleFocused = () => setFocused((prev) => !prev);
  const ref = useRef<HTMLDivElement>(null);

  useOnClickOutside(ref, () => setFocused(false));

  const [namespace, setNamespace] = useState<DropdownOption | undefined>(undefined);
  const [filters, setFilters] = useState<DropdownOption[]>([]);
  const [metrics, setMetrics] = useState<DropdownOption[]>([]);

  return (
    <RelativeContainer ref={ref}>
      <Button variant='secondary' style={{ textDecoration: 'none' }} onClick={toggleFocused}>
        <Image src='/icons/common/filter.svg' alt='filter' width={14} height={14} />
        <ButtonText>Filters</ButtonText>
        <Badge label={filters.length} filled />
      </Button>

      {focused && (
        <CardWrapper>
          <CardContent>
            <Dropdown title='Namespace' placeholder='Select namespace' options={[]} value={namespace} onSelect={(val) => setNamespace(val)} required />

            {/* TODO: make this a multi-select dropwdown (with internal checkboxes) */}
            <Dropdown title='Type' placeholder='All' options={[]} value={filters[0]} onSelect={(val) => setFilters((prev) => prev)} required />
            {/* TODO: make this a multi-select dropwdown (with internal checkboxes) */}
            <Dropdown title='Metric' placeholder='All' options={[]} value={metrics[0]} onSelect={(val) => setMetrics((prev) => prev)} required />
          </CardContent>
        </CardWrapper>
      )}
    </RelativeContainer>
  );
};

export { Filters };
