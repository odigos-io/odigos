import { useEffect, useState } from 'react';
import { useMutation, useQuery } from 'react-query';
import {
  getInstrumentationRules,
  getInstrumentationRule,
  createInstrumentationRule,
  updateInstrumentationRule,
  deleteInstrumentationRule,
} from '@/services';
import { InstrumentationRuleSpec } from '@/types';

export function useInstrumentationRules() {
  // Fetch all instrumentation rules
  const { isLoading, data, refetch } = useQuery<InstrumentationRuleSpec[]>(
    [],
    getInstrumentationRules
  );

  // State to manage sorted rules
  const [sortedRules, setSortedRules] = useState<
    InstrumentationRuleSpec[] | undefined
  >(undefined);

  // Mutations for create, update, and delete operations
  const { mutateAsync: createRule } = useMutation(
    (body: InstrumentationRuleSpec) => createInstrumentationRule(body)
  );

  const { mutateAsync: updateRule } = useMutation(
    (body: { id: string; data: InstrumentationRuleSpec }) =>
      updateInstrumentationRule(body.id, body.data)
  );

  const { mutateAsync: deleteRule } = useMutation((id: string) =>
    deleteInstrumentationRule(id)
  );

  // Set sorted rules when data changes
  useEffect(() => {
    setSortedRules(data || []);
  }, [data]);

  // Fetch rule by ID, refetch data if not available
  async function getRuleById(id: string) {
    let rules = data;
    if (!data) {
      const res = await refetch();
      rules = res.data;
    }
    return rules?.find((rule) => rule.ruleName === id); // Assuming id is ruleName, adjust as needed
  }

  // Function to sort rules by name or status
  function sortRules(condition: string) {
    const sorted = [...(data || [])].sort((a, b) => {
      switch (condition) {
        case 'NAME':
          return a.ruleName.localeCompare(b.ruleName);
        case 'STATUS':
          // Assuming 'disabled' is boolean; false is active, true is disabled
          const statusA = a.disabled ? 1 : -1;
          const statusB = b.disabled ? 1 : -1;
          return statusA - statusB;
        default:
          return 0;
      }
    });

    setSortedRules(sorted);
  }

  // Function to toggle the disabled status of a rule
  async function toggleRuleStatus(
    ids: string[],
    disabled: boolean
  ): Promise<boolean> {
    for (const id of ids) {
      const rule = await getRuleById(id);
      if (rule && rule.disabled !== disabled) {
        const body = {
          id,
          data: {
            ...rule,
            disabled,
          },
        };
        try {
          await updateRule(body);
        } catch (error) {
          return Promise.reject(false);
        }
      }
    }
    setTimeout(async () => {
      const res = await refetch();
      setSortedRules(res.data || []);
    }, 1000);

    return Promise.resolve(true);
  }

  // Function to handle refreshing the list of rules
  async function handleRulesRefresh() {
    const res = await refetch();
    setSortedRules(res.data || []);
  }

  // Create a new rule
  async function addRule(rule: InstrumentationRuleSpec) {
    try {
      await createRule(rule);
      await handleRulesRefresh();
    } catch (error) {
      console.error('Error creating rule:', error);
    }
  }

  // Delete a rule by ID
  async function removeRule(id: string) {
    try {
      await deleteRule(id);
      await handleRulesRefresh();
    } catch (error) {
      console.error('Error deleting rule:', error);
    }
  }

  return {
    isLoading,
    rules: sortedRules || [],
    addRule,
    updateRule,
    removeRule,
    sortRules,
    getRuleById,
    toggleRuleStatus,
    refetch: handleRulesRefresh,
  };
}
