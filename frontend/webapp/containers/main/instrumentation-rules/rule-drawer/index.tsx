import React, { useMemo, useState } from 'react';
import { RuleFormBody } from '../';
import buildCard from './build-card';
import styled from 'styled-components';
import { DataCard } from '@odigos/ui-components';
import { ACTION, DATA_CARDS, FORM_ALERTS } from '@/utils';
import OverviewDrawer from '../../overview/overview-drawer';
import { useDrawerStore, useNotificationStore } from '@odigos/ui-containers';
import { useInstrumentationRuleCRUD, useInstrumentationRuleFormData } from '@/hooks';
import { ENTITY_TYPES, getInstrumentationRuleIcon, INSTRUMENTATION_RULE_OPTIONS, NOTIFICATION_TYPE } from '@odigos/ui-utils';

interface Props {}

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;

export const RuleDrawer: React.FC<Props> = () => {
  const { addNotification } = useNotificationStore();
  const { drawerEntityId, setDrawerEntityId, setDrawerType } = useDrawerStore();

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useInstrumentationRuleFormData();
  const { instrumentationRules, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) {
        setDrawerType(null);
        setDrawerEntityId(null);
        resetFormData();
      }
    },
  });

  const thisItem = useMemo(() => {
    const found = instrumentationRules?.find((x) => x.ruleId === drawerEntityId);
    if (!!found) loadFormWithDrawerItem(found);

    return found;
  }, [instrumentationRules, drawerEntityId]);

  if (!thisItem) return null;

  const thisOptionType = INSTRUMENTATION_RULE_OPTIONS.find(({ type }) => type === thisItem.type);

  const handleEdit = (bool?: boolean) => {
    if (!thisItem.mutable && (bool || bool === undefined)) {
      addNotification({
        type: NOTIFICATION_TYPE.WARNING,
        title: FORM_ALERTS.FORBIDDEN,
        message: FORM_ALERTS.CANNOT_EDIT_RULE,
        crdType: ENTITY_TYPES.INSTRUMENTATION_RULE,
        target: drawerEntityId as string,
        hideFromHistory: true,
      });
    } else {
      setIsEditing(typeof bool === 'boolean' ? bool : true);
    }
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
    loadFormWithDrawerItem(thisItem);
  };

  const handleDelete = () => {
    if (!thisItem.mutable) {
      addNotification({
        type: NOTIFICATION_TYPE.WARNING,
        title: FORM_ALERTS.FORBIDDEN,
        message: FORM_ALERTS.CANNOT_DELETE_RULE,
        crdType: ENTITY_TYPES.INSTRUMENTATION_RULE,
        target: drawerEntityId as string,
        hideFromHistory: true,
      });
    } else {
      deleteInstrumentationRule(drawerEntityId as string);
    }
  };

  const handleSave = (newTitle: string) => {
    if (validateForm({ withAlert: true, alertTitle: ACTION.UPDATE })) {
      const title = newTitle !== thisItem.type ? newTitle : '';
      handleFormChange('ruleName', title);
      updateInstrumentationRule(drawerEntityId as string, { ...formData, ruleName: title });
    }
  };

  return (
    <OverviewDrawer
      title={thisItem.ruleName || thisItem.type}
      icon={getInstrumentationRuleIcon(thisItem.type)}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing && thisOptionType ? (
        <FormContainer>
          <RuleFormBody
            isUpdate
            rule={thisOptionType}
            formData={formData}
            formErrors={formErrors}
            handleFormChange={(...params) => {
              setIsFormDirty(true);
              handleFormChange(...params);
            }}
          />
        </FormContainer>
      ) : (
        <DataCard title={DATA_CARDS.RULE_DETAILS} data={!!thisItem ? buildCard(thisItem) : []} />
      )}
    </OverviewDrawer>
  );
};
