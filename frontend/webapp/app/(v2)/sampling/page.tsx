'use client';

import React, { useCallback, useMemo, useState } from 'react';
import { useQuery } from '@apollo/client';
import { GET_WORKLOADS } from '@/graphql';
import { useSamplingRuleCRUD } from '@/hooks';
import styled from 'styled-components';
import { FlexColumn, FlexRow, PageContent } from '@odigos/ui-kit/components';
import { BookIcon, PlusIcon, RefreshIcon, SamplingIcon } from '@odigos/ui-kit/icons';
import {
  PageTitle,
  SamplingRulesList,
  ViewSamplingRuleDrawer,
  CreateSamplingRuleDrawer,
  formStateToNoisyInput,
  formStateToHighlyRelevantInput,
  formStateToCostReductionInput,
  SamplingCategory,
  SAMPLING_SEGMENT_OPTIONS,
  SAMPLING_CATEGORY_NOTES,
  SAMPLING_CATEGORY_LIST_TITLES,
  CATEGORY_TO_RULE_CATEGORY,
  buildSamplingRuleItems,
  lookupViewRuleData,
  type ViewRuleData,
  type SamplingRuleItem,
  type SamplingRuleFormState,
} from '@odigos/ui-kit/containers/v2';
import { StatusType } from '@odigos/ui-kit/types';
import { DOCS_URL, PAGE_TITLE, PAGE_DESCRIPTION, BTN_SAMPLING_DOCS, BTN_REFRESH, BTN_CREATE_RULE } from './constants';
import { Button, ButtonSize, ButtonVariants, Note, Segment, SegmentVariant } from '@odigos/ui-kit/components/v2';

const Header = styled(FlexRow)`
  align-items: center;
  justify-content: space-between;
`;

export default function Page() {
  const {
    samplingRules,
    loading,
    fetchSamplingRules,
    createNoisyOperationRule,
    updateNoisyOperationRule,
    createHighlyRelevantOperationRule,
    updateHighlyRelevantOperationRule,
    createCostReductionRule,
    updateCostReductionRule,
    deleteNoisyOperationRule,
    deleteHighlyRelevantOperationRule,
    deleteCostReductionRule,
  } = useSamplingRuleCRUD();
  const { data: workloadsData } = useQuery<{ workloads: { id: { namespace: string; kind: string; name: string } }[] }>(GET_WORKLOADS);
  const [selectedCategory, setSelectedCategory] = useState<SamplingCategory>(SamplingCategory.Noisy);
  const [viewRuleData, setViewRuleData] = useState<ViewRuleData | null>(null);
  const [viewEditMode, setViewEditMode] = useState(false);
  const [isCreateOpen, setIsCreateOpen] = useState(false);

  const ruleItems = useMemo(() => buildSamplingRuleItems(samplingRules, selectedCategory), [samplingRules, selectedCategory]);

  const workloads = workloadsData?.workloads ?? [];

  const sourceOptions = useMemo(
    () => workloads.map(({ id: w }) => ({ id: `${w.namespace}/${w.kind}/${w.name}`, label: `${w.namespace} / ${w.kind} / ${w.name}` })),
    [workloads],
  );

  const namespaceOptions = useMemo(() => {
    const unique = Array.from(new Set(workloads.map(({ id: w }) => w.namespace))).sort();
    return unique.map((ns) => ({ id: ns, label: ns }));
  }, [workloads]);

  const handleDocs = useCallback(() => {
    window.open(DOCS_URL, '_blank', 'noopener,noreferrer');
  }, []);

  const handleCreateRule = useCallback(() => {
    setIsCreateOpen(true);
  }, []);

  const handleCloseCreateDrawer = useCallback(() => {
    setIsCreateOpen(false);
  }, []);

  const handleCreateSubmit = useCallback(
    (formState: SamplingRuleFormState) => {
      // Currently assumes a single sampling group; will become dynamic when multiple groups are supported.
      const samplingId = samplingRules[0]?.id ?? 'default';

      const category = CATEGORY_TO_RULE_CATEGORY[selectedCategory];
      switch (category) {
        case 'noisy':
          createNoisyOperationRule(samplingId, formStateToNoisyInput(formState));
          break;
        case 'highlyRelevant':
          createHighlyRelevantOperationRule(samplingId, formStateToHighlyRelevantInput(formState));
          break;
        case 'costReduction':
          createCostReductionRule(samplingId, formStateToCostReductionInput(formState));
          break;
      }

      setIsCreateOpen(false);
    },
    [samplingRules, selectedCategory, createNoisyOperationRule, createHighlyRelevantOperationRule, createCostReductionRule],
  );

  const handleRuleClick = useCallback(
    (item: SamplingRuleItem) => {
      const data = lookupViewRuleData(samplingRules, item);
      if (data) {
        setViewEditMode(false);
        setViewRuleData(data);
      }
    },
    [samplingRules],
  );

  const handleCloseDrawer = useCallback(() => {
    setViewRuleData(null);
    setViewEditMode(false);
  }, []);

  const handleEditRule = useCallback(
    (ruleId: string, samplingId: string) => {
      const existing = viewRuleData;
      if (existing && existing.rule.ruleId === ruleId && existing.samplingId === samplingId) {
        setViewEditMode(true);
        setViewRuleData({ ...existing });
      } else {
        for (const group of samplingRules) {
          if (group.id !== samplingId) continue;
          const all = [
            ...group.noisyOperations.map((r) => ({ category: 'noisy' as const, rule: r })),
            ...group.highlyRelevantOperations.map((r) => ({ category: 'highlyRelevant' as const, rule: r })),
            ...group.costReductionRules.map((r) => ({ category: 'costReduction' as const, rule: r })),
          ];
          const match = all.find((x) => x.rule.ruleId === ruleId);
          if (match) {
            setViewEditMode(true);
            setViewRuleData({ category: match.category, rule: match.rule, samplingId, summary: [] } as ViewRuleData);
            return;
          }
        }
      }
    },
    [viewRuleData, samplingRules],
  );

  const handleSaveEdit = useCallback(
    (formState: SamplingRuleFormState, ruleId: string, samplingId: string) => {
      const category = viewRuleData?.category;
      if (!category) return;

      switch (category) {
        case 'noisy':
          updateNoisyOperationRule(samplingId, ruleId, formStateToNoisyInput(formState));
          break;
        case 'highlyRelevant':
          updateHighlyRelevantOperationRule(samplingId, ruleId, formStateToHighlyRelevantInput(formState));
          break;
        case 'costReduction':
          updateCostReductionRule(samplingId, ruleId, formStateToCostReductionInput(formState));
          break;
      }

      setViewRuleData(null);
      setViewEditMode(false);
    },
    [viewRuleData, updateNoisyOperationRule, updateHighlyRelevantOperationRule, updateCostReductionRule],
  );

  const handleToggleDisabled = useCallback(
    (ruleId: string, samplingId: string, enabled: boolean) => {
      const category = viewRuleData?.category;
      if (!category) return;

      switch (category) {
        case 'noisy':
          updateNoisyOperationRule(samplingId, ruleId, { disabled: !enabled });
          break;
        case 'highlyRelevant':
          updateHighlyRelevantOperationRule(samplingId, ruleId, { disabled: !enabled });
          break;
        case 'costReduction':
          updateCostReductionRule(samplingId, ruleId, { disabled: !enabled, percentageAtMost: viewRuleData.rule.percentageAtMost });
          break;
      }
    },
    [viewRuleData, updateNoisyOperationRule, updateHighlyRelevantOperationRule, updateCostReductionRule],
  );

  const handleDeleteRule = useCallback(
    (ruleId: string, samplingId: string) => {
      const category = viewRuleData?.category;
      const cat = category || CATEGORY_TO_RULE_CATEGORY[selectedCategory];

      switch (cat) {
        case 'noisy':
          deleteNoisyOperationRule(samplingId, ruleId);
          break;
        case 'highlyRelevant':
          deleteHighlyRelevantOperationRule(samplingId, ruleId);
          break;
        case 'costReduction':
          deleteCostReductionRule(samplingId, ruleId);
          break;
      }

      setViewRuleData(null);
      setViewEditMode(false);
    },
    [viewRuleData, selectedCategory, deleteNoisyOperationRule, deleteHighlyRelevantOperationRule, deleteCostReductionRule],
  );

  return (
    <PageContent>
      <Header>
        <PageTitle icon={SamplingIcon} title={PAGE_TITLE} description={PAGE_DESCRIPTION} />

        <FlexRow $gap={8} $alignItems='center'>
          <Button label={BTN_SAMPLING_DOCS} leftIcon={BookIcon} size={ButtonSize.S} variant={ButtonVariants.Text} onClick={handleDocs} />
          <Button label={BTN_REFRESH} leftIcon={RefreshIcon} size={ButtonSize.S} variant={ButtonVariants.Text} onClick={fetchSamplingRules} loading={loading} />
          <Button label={BTN_CREATE_RULE} rightIcon={PlusIcon} size={ButtonSize.S} variant={ButtonVariants.Primary} onClick={handleCreateRule} />
        </FlexRow>
      </Header>

      <FlexColumn $gap={12} $alignItems='flex-start'>
        <Segment variant={SegmentVariant.Underline} options={SAMPLING_SEGMENT_OPTIONS} selected={selectedCategory} setSelected={setSelectedCategory} />
        <Note status={StatusType.Default} message={SAMPLING_CATEGORY_NOTES[selectedCategory]} />
      </FlexColumn>

      <SamplingRulesList
        title={SAMPLING_CATEGORY_LIST_TITLES[selectedCategory]}
        items={ruleItems}
        isLoading={loading}
        onRuleClick={handleRuleClick}
        onEditRule={handleEditRule}
        onDeleteRule={handleDeleteRule}
      />

      <ViewSamplingRuleDrawer
        data={viewRuleData}
        defaultEditMode={viewEditMode}
        onClose={handleCloseDrawer}
        onDelete={handleDeleteRule}
        onToggleDisabled={handleToggleDisabled}
        onSaveEdit={handleSaveEdit}
        sourceOptions={sourceOptions}
        namespaceOptions={namespaceOptions}
      />

      <CreateSamplingRuleDrawer isOpen={isCreateOpen} category={CATEGORY_TO_RULE_CATEGORY[selectedCategory]} onClose={handleCloseCreateDrawer} onSubmit={handleCreateSubmit} sourceOptions={sourceOptions} namespaceOptions={namespaceOptions} />
    </PageContent>
  );
}
