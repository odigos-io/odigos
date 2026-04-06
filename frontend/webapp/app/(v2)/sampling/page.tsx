'use client';

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import styled from 'styled-components';
import { GET_WORKLOADS } from '@/graphql';
import { useQuery } from '@apollo/client';
import { useSamplingRuleCRUD } from '@/hooks';
import { StatusType } from '@odigos/ui-kit/types';
import { RichTitle } from '@odigos/ui-kit/snippets/v2';
import { PlusIcon, RefreshIcon, SamplingIcon } from '@odigos/ui-kit/icons';
import { FlexColumn, FlexRow, PageContent } from '@odigos/ui-kit/components';
import { Button, ButtonSize, ButtonVariants, Note, Segment, SegmentVariant, WarningModal } from '@odigos/ui-kit/components/v2';
import {
  PAGE_TITLE,
  PAGE_DESCRIPTION,
  BTN_REFRESH,
  BTN_CREATE_RULE,
  DELETE_MODAL_TITLE,
  DELETE_MODAL_DESCRIPTION,
  DELETE_MODAL_APPROVE,
  DELETE_MODAL_CANCEL,
  AUTO_RULE_TITLE,
  HIGHLY_RELEVANT_AUTO_RULE_TITLE,
  COST_REDUCTION_AUTO_RULE_TITLE,
  DUPLICATE_RULE_WARNING,
} from './constants';
import {
  AutoRuleCard,
  buildAutoRuleSummary,
  findHighlyRelevantAutoRule,
  buildHighlyRelevantAutoRuleSummary,
  findCostReductionAutoRule,
  buildCostReductionAutoRuleSummary,
  EditAutoRuleDrawer,
  EditHighlyRelevantAutoRuleDrawer,
  EditCostReductionAutoRuleDrawer,
  SamplingRulesList,
  ViewSamplingRuleDrawer,
  CreateSamplingRuleDrawer,
  formStateToNoisyInput,
  formStateToHighlyRelevantInput,
  formStateToCostReductionInput,
  findDuplicateRuleId,
  type DuplicateValidationResult,
  SamplingCategory,
  SAMPLING_SEGMENT_OPTIONS,
  SAMPLING_CATEGORY_NOTES,
  SAMPLING_CATEGORY_LIST_TITLES,
  CATEGORY_TO_RULE_CATEGORY,
  buildSamplingRuleItems,
  buildSummaryForRule,
  refreshViewRuleData,
  lookupViewRuleData,
  type ViewRuleData,
  type SamplingRuleItem,
  type SamplingRuleFormState,
} from '@odigos/ui-kit/containers/v2';

const Header = styled(FlexRow)`
  align-items: center;
  justify-content: space-between;
`;

export default function Page() {
  const {
    samplingRules,
    k8sHealthProbesConfig,
    loading,
    fetchSamplingRules,
    createNoisyOperationRule,
    updateNoisyOperationRule,
    createHighlyRelevantOperationRule,
    updateHighlyRelevantOperationRule,
    createCostReductionRule,
    updateCostReductionRule,
    deleteSamplingRule,
    updateK8sHealthProbesConfig,
  } = useSamplingRuleCRUD();
  const { data: workloadsData } = useQuery<{ workloads: { id: { namespace: string; kind: string; name: string } }[] }>(GET_WORKLOADS, {
    variables: { filter: { markedForInstrumentation: true } },
  });
  const [selectedCategory, setSelectedCategory] = useState<SamplingCategory>(SamplingCategory.Noisy);
  const [viewRuleData, setViewRuleData] = useState<ViewRuleData | null>(null);
  const [viewEditMode, setViewEditMode] = useState(false);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ ruleId: string; samplingId: string } | null>(null);
  const [isAutoRuleDrawerOpen, setIsAutoRuleDrawerOpen] = useState(false);
  const [isHRAutoRuleDrawerOpen, setIsHRAutoRuleDrawerOpen] = useState(false);
  const [isCRAutoRuleDrawerOpen, setIsCRAutoRuleDrawerOpen] = useState(false);

  const autoRuleSummary = useMemo(() => buildAutoRuleSummary(k8sHealthProbesConfig), [k8sHealthProbesConfig]);

  const highlyRelevantAutoRule = useMemo(() => findHighlyRelevantAutoRule(samplingRules), [samplingRules]);
  const highlyRelevantAutoSummary = useMemo(() => buildHighlyRelevantAutoRuleSummary(highlyRelevantAutoRule?.rule ?? null), [highlyRelevantAutoRule]);

  const costReductionAutoRule = useMemo(() => findCostReductionAutoRule(samplingRules), [samplingRules]);
  const costReductionAutoSummary = useMemo(() => buildCostReductionAutoRuleSummary(costReductionAutoRule?.rule ?? null), [costReductionAutoRule]);

  const viewRuleRef = useRef(viewRuleData);
  viewRuleRef.current = viewRuleData;

  useEffect(() => {
    const current = viewRuleRef.current;
    if (!current) return;

    const refreshed = refreshViewRuleData(samplingRules, current);
    if (refreshed) {
      setViewRuleData(refreshed);
    } else {
      setViewRuleData(null);
      setViewEditMode(false);
    }
  }, [samplingRules]);

  const excludeRuleIds = useMemo(() => {
    const ids = new Set<string>();
    if (highlyRelevantAutoRule) ids.add(highlyRelevantAutoRule.rule.ruleId);
    if (costReductionAutoRule) ids.add(costReductionAutoRule.rule.ruleId);
    return ids;
  }, [highlyRelevantAutoRule, costReductionAutoRule]);

  const ruleItems = useMemo(() => buildSamplingRuleItems(samplingRules, selectedCategory, excludeRuleIds), [samplingRules, selectedCategory, excludeRuleIds]);

  const workloads = workloadsData?.workloads ?? [];

  const sourceOptions = useMemo(() => workloads.map(({ id: w }) => ({ id: `${w.namespace}/${w.kind}/${w.name}`, label: `${w.namespace} / ${w.kind} / ${w.name}` })), [workloads]);

  const namespaceOptions = useMemo(() => {
    const unique = Array.from(new Set(workloads.map(({ id: w }) => w.namespace))).sort();
    return unique.map((ns) => ({ id: ns, label: ns }));
  }, [workloads]);

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
            setViewRuleData({ category: match.category, rule: match.rule, samplingId, summary: buildSummaryForRule(match.category, match.rule) } as ViewRuleData);
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

      setViewEditMode(false);
    },
    [viewRuleData, updateNoisyOperationRule, updateHighlyRelevantOperationRule, updateCostReductionRule],
  );

  const handleDeleteRule = useCallback((ruleId: string, samplingId: string) => {
    setDeleteTarget({ ruleId, samplingId });
  }, []);

  const confirmDelete = useCallback(() => {
    if (!deleteTarget) return;
    const { ruleId, samplingId } = deleteTarget;
    const cat = viewRuleData?.category || CATEGORY_TO_RULE_CATEGORY[selectedCategory];

    deleteSamplingRule(samplingId, ruleId, cat);
    setViewRuleData(null);
    setViewEditMode(false);
    setDeleteTarget(null);
  }, [deleteTarget, viewRuleData, selectedCategory, deleteSamplingRule]);

  const handleCancelDelete = useCallback(() => {
    setDeleteTarget(null);
  }, []);

  const handleEditAutoRule = useCallback(() => {
    setIsAutoRuleDrawerOpen(true);
  }, []);

  const handleCloseAutoRuleDrawer = useCallback(() => {
    setIsAutoRuleDrawerOpen(false);
  }, []);

  const handleSaveAutoRule = useCallback(
    (enabled: boolean, keepPercentage: number) => {
      updateK8sHealthProbesConfig(enabled, keepPercentage);
      setIsAutoRuleDrawerOpen(false);
    },
    [updateK8sHealthProbesConfig],
  );

  const handleEditHRAutoRule = useCallback(() => {
    setIsHRAutoRuleDrawerOpen(true);
  }, []);

  const handleCloseHRAutoRuleDrawer = useCallback(() => {
    setIsHRAutoRuleDrawerOpen(false);
  }, []);

  const handleSaveHRAutoRule = useCallback(
    (enabled: boolean) => {
      const samplingId = samplingRules[0]?.id ?? 'default';
      const existing = highlyRelevantAutoRule;

      if (existing) {
        updateHighlyRelevantOperationRule(existing.samplingId, existing.rule.ruleId, {
          name: existing.rule.name,
          disabled: !enabled,
          error: true,
          sourceScopes: [],
          operation: null,
          percentageAtLeast: null,
          notes: existing.rule.notes,
        });
      } else if (enabled) {
        createHighlyRelevantOperationRule(samplingId, {
          name: 'Auto - Keep All Error Traces',
          disabled: false,
          error: true,
          sourceScopes: [],
          operation: null,
          percentageAtLeast: null,
        });
      }

      setIsHRAutoRuleDrawerOpen(false);
    },
    [samplingRules, highlyRelevantAutoRule, createHighlyRelevantOperationRule, updateHighlyRelevantOperationRule],
  );

  const handleEditCRAutoRule = useCallback(() => {
    setIsCRAutoRuleDrawerOpen(true);
  }, []);

  const handleCloseCRAutoRuleDrawer = useCallback(() => {
    setIsCRAutoRuleDrawerOpen(false);
  }, []);

  const handleSaveCRAutoRule = useCallback(
    (enabled: boolean, dropPercentage: number) => {
      const samplingId = samplingRules[0]?.id ?? 'default';
      const existing = costReductionAutoRule;

      if (existing) {
        updateCostReductionRule(existing.samplingId, existing.rule.ruleId, {
          name: existing.rule.name,
          disabled: !enabled,
          sourceScopes: [],
          operation: null,
          percentageAtMost: dropPercentage,
          notes: existing.rule.notes,
        });
      } else if (enabled) {
        createCostReductionRule(samplingId, {
          name: 'Auto - Drop Traces Cluster-Wide',
          disabled: false,
          sourceScopes: [],
          operation: null,
          percentageAtMost: dropPercentage,
        });
      }

      setIsCRAutoRuleDrawerOpen(false);
    },
    [samplingRules, costReductionAutoRule, createCostReductionRule, updateCostReductionRule],
  );

  const validateCreateForm = useCallback(
    (formState: SamplingRuleFormState): DuplicateValidationResult | null => {
      const category = CATEGORY_TO_RULE_CATEGORY[selectedCategory];
      let dupId: string | null = null;
      switch (category) {
        case 'noisy': {
          const input = formStateToNoisyInput(formState);
          dupId = findDuplicateRuleId(samplingRules, category, { sourceScopes: input.sourceScopes, operation: input.operation });
          break;
        }
        case 'highlyRelevant': {
          const input = formStateToHighlyRelevantInput(formState);
          dupId = findDuplicateRuleId(samplingRules, category, {
            sourceScopes: input.sourceScopes,
            operation: input.operation,
            error: input.error ?? false,
            durationAtLeastMs: input.durationAtLeastMs,
          });
          break;
        }
        case 'costReduction': {
          const input = formStateToCostReductionInput(formState);
          dupId = findDuplicateRuleId(samplingRules, category, { sourceScopes: input.sourceScopes, operation: input.operation });
          break;
        }
      }
      return dupId ? { message: DUPLICATE_RULE_WARNING, ruleId: dupId } : null;
    },
    [samplingRules, selectedCategory],
  );

  const validateEditForm = useCallback(
    (formState: SamplingRuleFormState): DuplicateValidationResult | null => {
      const category = viewRuleData?.category;
      const excludeId = viewRuleData?.rule.ruleId;
      if (!category) return null;

      let dupId: string | null = null;
      switch (category) {
        case 'noisy': {
          const input = formStateToNoisyInput(formState);
          dupId = findDuplicateRuleId(samplingRules, category, { sourceScopes: input.sourceScopes, operation: input.operation }, excludeId);
          break;
        }
        case 'highlyRelevant': {
          const input = formStateToHighlyRelevantInput(formState);
          dupId = findDuplicateRuleId(
            samplingRules,
            category,
            { sourceScopes: input.sourceScopes, operation: input.operation, error: input.error ?? false, durationAtLeastMs: input.durationAtLeastMs },
            excludeId,
          );
          break;
        }
        case 'costReduction': {
          const input = formStateToCostReductionInput(formState);
          dupId = findDuplicateRuleId(samplingRules, category, { sourceScopes: input.sourceScopes, operation: input.operation }, excludeId);
          break;
        }
      }
      return dupId ? { message: DUPLICATE_RULE_WARNING, ruleId: dupId } : null;
    },
    [samplingRules, viewRuleData],
  );

  const handleNavigateToDuplicate = useCallback(
    (ruleId: string) => {
      setIsCreateOpen(false);
      const category = CATEGORY_TO_RULE_CATEGORY[selectedCategory];
      const samplingId = samplingRules[0]?.id ?? 'default';

      for (const group of samplingRules) {
        const all = [
          ...group.noisyOperations.map((r) => ({ category: 'noisy' as const, rule: r })),
          ...group.highlyRelevantOperations.map((r) => ({ category: 'highlyRelevant' as const, rule: r })),
          ...group.costReductionRules.map((r) => ({ category: 'costReduction' as const, rule: r })),
        ];
        const match = all.find((x) => x.rule.ruleId === ruleId);
        if (match) {
          setViewEditMode(true);
          setViewRuleData({ category: match.category, rule: match.rule, samplingId: group.id, summary: buildSummaryForRule(match.category, match.rule) } as ViewRuleData);
          return;
        }
      }
    },
    [samplingRules, selectedCategory],
  );

  return (
    <PageContent>
      <Header>
        <RichTitle icon={SamplingIcon} title={PAGE_TITLE} subTitle={PAGE_DESCRIPTION} />

        <FlexRow $gap={8} $alignItems='center'>
          <Button label={BTN_REFRESH} leftIcon={RefreshIcon} size={ButtonSize.S} variant={ButtonVariants.Text} onClick={fetchSamplingRules} loading={loading} />
          <Button label={BTN_CREATE_RULE} rightIcon={PlusIcon} size={ButtonSize.S} variant={ButtonVariants.Primary} onClick={handleCreateRule} />
        </FlexRow>
      </Header>

      <FlexColumn $gap={12} $alignItems='flex-start'>
        <Segment variant={SegmentVariant.Underline} options={SAMPLING_SEGMENT_OPTIONS} selected={selectedCategory} setSelected={setSelectedCategory} />
        <Note status={StatusType.Default} message={SAMPLING_CATEGORY_NOTES[selectedCategory]} />
      </FlexColumn>

      {selectedCategory === SamplingCategory.Noisy && <AutoRuleCard title={AUTO_RULE_TITLE} summary={autoRuleSummary} onEdit={handleEditAutoRule} />}
      {selectedCategory === SamplingCategory.HighlyRelevant && <AutoRuleCard title={HIGHLY_RELEVANT_AUTO_RULE_TITLE} summary={highlyRelevantAutoSummary} onEdit={handleEditHRAutoRule} />}
      {selectedCategory === SamplingCategory.CostReduction && <AutoRuleCard title={COST_REDUCTION_AUTO_RULE_TITLE} summary={costReductionAutoSummary} onEdit={handleEditCRAutoRule} />}

      <SamplingRulesList
        title={SAMPLING_CATEGORY_LIST_TITLES[selectedCategory]}
        items={ruleItems}
        isLoading={loading}
        showTypeFilter={selectedCategory === SamplingCategory.HighlyRelevant}
        onRuleClick={handleRuleClick}
        onEditRule={handleEditRule}
        onDeleteRule={handleDeleteRule}
      />

      <ViewSamplingRuleDrawer
        data={viewRuleData}
        defaultEditMode={viewEditMode}
        onClose={handleCloseDrawer}
        onDelete={handleDeleteRule}
        onSaveEdit={handleSaveEdit}
        sourceOptions={sourceOptions}
        namespaceOptions={namespaceOptions}
        validateForm={validateEditForm}
        onNavigateToDuplicate={handleNavigateToDuplicate}
      />

      <CreateSamplingRuleDrawer
        isOpen={isCreateOpen}
        category={CATEGORY_TO_RULE_CATEGORY[selectedCategory]}
        onClose={handleCloseCreateDrawer}
        onSubmit={handleCreateSubmit}
        sourceOptions={sourceOptions}
        namespaceOptions={namespaceOptions}
        validateForm={validateCreateForm}
        onNavigateToDuplicate={handleNavigateToDuplicate}
      />

      <EditAutoRuleDrawer
        isOpen={isAutoRuleDrawerOpen}
        enabled={k8sHealthProbesConfig?.enabled ?? false}
        keepPercentage={k8sHealthProbesConfig?.keepPercentage ?? 0}
        onClose={handleCloseAutoRuleDrawer}
        onSave={handleSaveAutoRule}
      />

      <EditHighlyRelevantAutoRuleDrawer
        isOpen={isHRAutoRuleDrawerOpen}
        enabled={!!highlyRelevantAutoRule && !highlyRelevantAutoRule.rule.disabled}
        onClose={handleCloseHRAutoRuleDrawer}
        onSave={handleSaveHRAutoRule}
      />

      <EditCostReductionAutoRuleDrawer
        isOpen={isCRAutoRuleDrawerOpen}
        enabled={!!costReductionAutoRule && !costReductionAutoRule.rule.disabled}
        dropPercentage={costReductionAutoRule?.rule.percentageAtMost ?? 25}
        onClose={handleCloseCRAutoRuleDrawer}
        onSave={handleSaveCRAutoRule}
      />

      <WarningModal
        title={DELETE_MODAL_TITLE}
        description={DELETE_MODAL_DESCRIPTION}
        isOpen={!!deleteTarget}
        onClose={handleCancelDelete}
        onApprove={confirmDelete}
        approveLabel={DELETE_MODAL_APPROVE}
        denyLabel={DELETE_MODAL_CANCEL}
      />
    </PageContent>
  );
}
