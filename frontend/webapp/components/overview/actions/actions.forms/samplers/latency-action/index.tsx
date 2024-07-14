import React, { useState, useEffect } from 'react';
import { useSources } from '@/hooks';
import styled from 'styled-components';
import {
  KeyvalButton,
  KeyvalDropDown,
  KeyvalInput,
  KeyvalLink,
  KeyvalText,
} from '@/design.system';
import theme from '@/styles/palette';
import { Tooltip } from '@keyval-dev/design-system';

const FormWrapper = styled.div`
  width: fit-content;
`;

const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
`;

const TableHeader = styled.th`
  text-align: left;
  padding: 8px;
`;

const TableCell = styled.td`
  width: 220px;
  padding-bottom: 8px;
`;

interface HttpRouteFilter {
  fallback_sampling_ratio?: number;
  minimum_latency_threshold?: number;
  http_route?: string;
  service_name?: string;
}

interface LatencySampler {
  endpoints_filters: HttpRouteFilter[];
}

interface LatencySamplerFormProps {
  data: LatencySampler;
  onChange: (key: string, value: LatencySampler | null) => void;
}

const ACTION_DATA_KEY = 'actionData';

export function LatencySamplerForm({
  data,
  onChange,
}: LatencySamplerFormProps): React.JSX.Element {
  const { sources } = useSources();
  const [filters, setFilters] = useState<HttpRouteFilter[]>(
    data?.endpoints_filters.length ? data.endpoints_filters : [{}]
  );

  useEffect(() => {
    if (filters.length === 0) {
      setFilters([{}]);
    }
  }, [filters]);

  const memoizedSources = React.useMemo(() => {
    return sources.map((source, index) => ({
      id: index,
      label: source.name,
    }));
  }, [sources]);

  function handleOnChange(index: number, key: string, value: any): void {
    const updatedFilters = filters.map((filter, i) =>
      i === index ? { ...filter, [key]: value } : filter
    );
    setFilters(updatedFilters);
    onChange(ACTION_DATA_KEY, { endpoints_filters: updatedFilters });
  }

  function handleAddFilter(): void {
    setFilters([...filters, {}]);
  }

  function handleRemoveFilter(index: number): void {
    const updatedFilters = filters.filter((_, i) => i !== index);
    setFilters(updatedFilters);
    onChange(ACTION_DATA_KEY, { endpoints_filters: updatedFilters });
  }

  function getDropdownValue(serviceName: string): {
    id: number;
    label: string;
  } {
    return (
      memoizedSources.find((source) => source.label === serviceName) || {
        id: 0,
        label: '',
      }
    );
  }

  return (
    <FormWrapper>
      <Table>
        <thead>
          <tr>
            <TableHeader>
              <KeyvalText size={12}>Service Name</KeyvalText>
            </TableHeader>
            <TableHeader>
              <KeyvalText size={12}>Http Route</KeyvalText>
            </TableHeader>
            <TableHeader>
              <KeyvalText size={12}>Minimum Latency Threshold (ms)</KeyvalText>
            </TableHeader>
            <TableHeader>
              <Tooltip text="allows you to retain a specified percentage of traces that fall below the threshold">
                <KeyvalText size={12}>Fallback Sampling Ratio </KeyvalText>
              </Tooltip>
            </TableHeader>
            <TableHeader></TableHeader>
          </tr>
        </thead>
        <tbody>
          {filters.map((filter, index) => (
            <tr key={index}>
              <TableCell>
                <KeyvalDropDown
                  width={198}
                  data={memoizedSources}
                  value={getDropdownValue(filter.service_name || '')}
                  onChange={(value) =>
                    handleOnChange(index, 'service_name', value.label)
                  }
                />
              </TableCell>
              <TableCell>
                <KeyvalInput
                  style={{ width: 192, height: 39 }}
                  value={filter.http_route || ''}
                  onChange={(value) =>
                    handleOnChange(index, 'http_route', value)
                  }
                  placeholder="e.g. /api/v1/users"
                  type="text"
                />
              </TableCell>
              <TableCell>
                <KeyvalInput
                  style={{ width: 192, height: 39 }}
                  value={filter.minimum_latency_threshold?.toString() || ''}
                  onChange={(value) =>
                    handleOnChange(index, 'minimum_latency_threshold', +value)
                  }
                  placeholder="e.g. 1000"
                  type="number"
                  min={0}
                />
              </TableCell>
              <TableCell>
                <KeyvalInput
                  style={{ width: 192, height: 39 }}
                  value={filter.fallback_sampling_ratio?.toString() || ''}
                  onChange={(value) =>
                    handleOnChange(index, 'fallback_sampling_ratio', +value)
                  }
                  placeholder="e.g. 20"
                  type="number"
                  min={0}
                  max={100}
                />
              </TableCell>
              <TableCell>
                {filters.length > 1 && (
                  <KeyvalLink
                    value="remove"
                    fontSize={12}
                    onClick={() => handleRemoveFilter(index)}
                  ></KeyvalLink>
                )}
              </TableCell>
            </tr>
          ))}
        </tbody>
      </Table>
      <KeyvalButton
        onClick={handleAddFilter}
        style={{ height: 32, width: 140, marginTop: 8 }}
      >
        <KeyvalText size={14} weight={600} color={theme.text.dark_button}>
          {'+ Add Filter'}
        </KeyvalText>
      </KeyvalButton>
    </FormWrapper>
  );
}
