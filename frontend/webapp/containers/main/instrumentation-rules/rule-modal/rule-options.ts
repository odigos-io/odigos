import { InstrumentationRuleType } from '@/types';
import { getRuleIcon } from '@/utils';

export type RuleOption = {
  id: string;
  label: string;
  type?: InstrumentationRuleType;
  icon?: string;
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
];
