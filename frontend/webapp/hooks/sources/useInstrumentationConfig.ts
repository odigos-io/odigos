import { useEffect, useState } from 'react';
import { InstrumentationConfig } from '@/types';
export const FAKE_LIST: InstrumentationConfig[] = [
  {
    option_key: 'option_8',
    option_value_boolean: true,
    span_kind: 'Internal',
    instrumentation_libraries: [
      {
        instrumentation_library_name: 'libA',
        language: 'Java',
        selected: true,
      },
    ],
  },
  {
    option_key: 'option_90',
    option_value_boolean: true,
    span_kind: 'Consumer',
    instrumentation_libraries: [
      {
        instrumentation_library_name: 'libB',
        language: 'Python',
        selected: true,
      },
    ],
  },
];

export function useInstrumentationConfig() {
  const [config, setConfig] = useState<InstrumentationConfig[]>(FAKE_LIST);

  return {};
}
