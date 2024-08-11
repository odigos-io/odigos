import React from 'react';
import {useRouter} from 'next/navigation';
import {NewActionCard} from '@/components';
import {ActionItemCard, ActionsType} from '@/types';
import {KeyvalLink, KeyvalText} from '@/design.system';
import {ACTION, ACTION_DOCS_LINK, OVERVIEW} from '@/utils';
import {ActionCardWrapper, ActionsListWrapper, DescriptionWrapper, LinkWrapper,} from './styled';

const ITEMS = [
    {
        id: 'add_cluster_info',
        title: 'Add Cluster Info',
        description: 'Add static cluster-scoped attributes to your data.',
        type: ActionsType.ADD_CLUSTER_INFO,
        icon: ActionsType.ADD_CLUSTER_INFO,
    },
    {
        id: 'delete_attribute',
        title: 'Delete Attribute',
        description: 'Delete attributes from logs, metrics, and traces.',
        type: ActionsType.DELETE_ATTRIBUTES,
        icon: ActionsType.DELETE_ATTRIBUTES,
    },
    {
        id: 'rename_attribute',
        title: 'Rename Attribute',
        description: 'Rename attributes in logs, metrics, and traces.',
        type: ActionsType.RENAME_ATTRIBUTES,
        icon: ActionsType.RENAME_ATTRIBUTES,
    },
    {
        id: 'error-sampler',
        title: 'Error Sampler',
        description: 'Sample errors based on percentage.',
        type: ActionsType.ERROR_SAMPLER,
        icon: ActionsType.ERROR_SAMPLER,
    },
    {
        id: 'probabilistic-sampler',
        title: 'Probabilistic Sampler',
        description: 'Sample traces based on percentage.',
        type: ActionsType.PROBABILISTIC_SAMPLER,
        icon: ActionsType.PROBABILISTIC_SAMPLER,
    },
    {
        id: 'latency-action',
        title: 'Latency Action',
        description: 'Add latency to your traces.',
        type: ActionsType.LATENCY_SAMPLER,
        icon: ActionsType.LATENCY_SAMPLER,
    },
    {
        id: 'pii-masking',
        title: 'PII Masking',
        description: 'Mask PII data in your traces.',
        type: ActionsType.PII_MASKING,
        icon: ActionsType.PII_MASKING,
    },
];

export function ChooseActionContainer(): React.JSX.Element {
    const router = useRouter();

    function onItemClick({item}: { item: ActionItemCard }) {
        router.push(`/create-action?type=${item.type}`);
    }

    function renderActionsList() {
        return ITEMS.map((item) => {
            return (
                <ActionCardWrapper data-cy={'choose-action-' + item.type} key={item.id}>
                    <NewActionCard item={item} onClick={onItemClick}/>
                </ActionCardWrapper>
            );
        });
    }

    return (
        <>
            <DescriptionWrapper>
                <KeyvalText size={14}>{OVERVIEW.ACTION_DESCRIPTION}</KeyvalText>
                <LinkWrapper>
                    <KeyvalLink
                        fontSize={14}
                        value={ACTION.LINK_TO_DOCS}
                        onClick={() => window.open(ACTION_DOCS_LINK, '_blank')}
                    />
                </LinkWrapper>
            </DescriptionWrapper>
            <ActionsListWrapper>{renderActionsList()}</ActionsListWrapper>
        </>
    );
}
