import React, { useEffect, useState } from 'react';
import { YMLEditor } from '@/design.system';
import { ActionState } from '@/types';

export default function AddClusterInfoYaml({
  data,
  onChange,
}: {
  data: ActionState;
  onChange: (key: string, value: any) => void;
}) {
  const [yaml, setYaml] = useState({});

  useEffect(() => {
    if (data.actionName && data.actionName.endsWith(' ')) {
      return;
    }

    if (data.actionNote && data.actionNote.endsWith(' ')) {
      return;
    }

    const clusterAttributes = data.actionData?.clusterAttributes.map(
      (attr) => ({
        [attr.attributeName]: attr.attributeStringValue,
      })
    );

    const newYaml = {
      apiVersion: 'actions.odigos.io/v1alpha1',
      kind: 'AddClusterInfo',
      metadata: {
        generateName: 'da-',
        namespace: 'odigos-system',
      },
      spec: {
        actionName: data.actionName || undefined,
        clusterAttributes,
        signals: data.selectedMonitors
          .filter((m) => m.checked)
          .map((m) => m.label.toUpperCase()),
        actionNote: data.actionNote || undefined,
      },
    };

    setYaml(newYaml);
  }, [data]);

  if (Object.keys(yaml).length === 0) {
    return null;
  }

  return (
    <>
      <YMLEditor data={yaml} setData={() => {}} />
    </>
  );
}
