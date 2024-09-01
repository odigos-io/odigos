import React from 'react';

import { KeyvalText } from '@/design.system';
import { InstrumentationConfig } from '@/types';
import { TextInformationWrapper } from './style.styled';
import InstrumentationConfigListItem from './instrumentation-config-list-item';
import { FAKE_LIST } from './fake';
const SOURCES = {
  INSTRUMENTATION_CONFIG_LIBRARIES: 'Data Collection Settings',
  INSTRUMENTATION_CONFIG_INFO:
    'Boost your trace data with precision. Every selection from these libraries contributes essential details to the data sent to the chosen destinations, empowering you with richer insights.',
};

interface InstrumentationConfigListProps {
  list: InstrumentationConfig[] | undefined;
  onChange: (list: InstrumentationConfig[]) => void;
}

export function InstrumentationConfigList({
  list = FAKE_LIST,
  onChange,
}: InstrumentationConfigListProps) {
  function onOptionChange(option: InstrumentationConfig) {
    const newConfig = list?.map((item: InstrumentationConfig) => {
      if (
        item.optionKey === option.optionKey &&
        item.spanKind === option.spanKind
      ) {
        return {
          ...option,
        };
      }
      return item;
    });

    onChange(newConfig || []);
  }

  return (
    <div style={{ maxWidth: 500 }}>
      <TextInformationWrapper>
        <KeyvalText weight={600} size={18}>
          {SOURCES.INSTRUMENTATION_CONFIG_LIBRARIES}
        </KeyvalText>
        <KeyvalText style={{ lineHeight: '1.3' }} size={14}>
          {SOURCES.INSTRUMENTATION_CONFIG_INFO}
        </KeyvalText>
      </TextInformationWrapper>
      {list?.map((item: InstrumentationConfig, index: number) => (
        <div key={index}>
          <InstrumentationConfigListItem
            item={item}
            onOptionChange={onOptionChange}
          />
        </div>
      ))}
    </div>
  );
}
