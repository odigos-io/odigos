import { getRuleIcon } from '@/utils';
import { Types } from '@odigos/ui-components';
import { InstrumentationRuleType } from '@/types';

export type RuleOption = {
  id: string;
  label: string;
  type?: InstrumentationRuleType;
  icon?: Types.SVG;
  description?: string;
  docsEndpoint?: string;
  docsDescription?: string;
  items?: RuleOption[];
};

export const RULE_OPTIONS: RuleOption[] = [
  {
    id: 'payload_collection',
    label: 'Payload Collection',
    description: 'Collect span attributes containing payload data to traces.',
    type: InstrumentationRuleType.PAYLOAD_COLLECTION,
    icon: getRuleIcon(InstrumentationRuleType.PAYLOAD_COLLECTION),
    docsEndpoint: '/pipeline/rules/payloadcollection',
    docsDescription: 'The “Payload Collection” Rule can be used to add span attributes containing payload data to traces.',
  },
  {
    id: 'code_attributes',
    label: 'Code Attributes',
    description: 'Collect code attributes containing payload data to traces.',
    type: InstrumentationRuleType.CODE_ATTRIBUTES,
    icon: getRuleIcon(InstrumentationRuleType.CODE_ATTRIBUTES),
    docsEndpoint: '/pipeline/rules/codeattributes',
    docsDescription: 'The “Code Attributes” Rule can be used to add code attributes containing payload data to traces.',
  },
];
