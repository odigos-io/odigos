import React, { useEffect, useState } from 'react';
import { useInstrumentationRules } from '@/hooks';
import theme from '@/styles/palette';
import { useRouter } from 'next/navigation';
import { EmptyList } from '@/components';
import {
  KeyvalText,
  KeyvalButton,
  KeyvalLoader,
  KeyvalSearchInput,
} from '@/design.system';
import {
  InstrumentationRulesContainer,
  Container,
  Content,
  Header,
  HeaderRight,
} from './styled';
import { InstrumentationRulesTable } from '@/components/overview/instrumentation-rules/rules-table';

export function ManagedInstrumentationRulesContainer() {
  const [searchInput, setSearchInput] = useState('');
  const router = useRouter();

  const { isLoading, rules, sortRules, refetch } = useInstrumentationRules();

  useEffect(() => {
    refetch();
  }, []);

  function handleAddRule() {
    router.push('/choose-rule');
  }

  function handleEditRule(id: string) {
    router.push(`edit-rule?id=${id}`);
  }

  function filterRules() {
    return rules;
  }

  if (isLoading) return <KeyvalLoader />;

  return (
    <>
      <Container>
        {!rules?.length ? (
          <EmptyList
            title={'No rules found'}
            btnTitle={'Add Rule'}
            btnAction={handleAddRule}
          />
        ) : (
          <InstrumentationRulesContainer>
            <Header>
              <KeyvalSearchInput
                containerStyle={{ padding: '6px 8px' }}
                placeholder={'Search Rule'}
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
              />
              <HeaderRight>
                <KeyvalButton onClick={handleAddRule} style={{ height: 32 }}>
                  <KeyvalText
                    size={14}
                    weight={600}
                    color={theme.text.dark_button}
                  >
                    {'Add Rule'}
                  </KeyvalText>
                </KeyvalButton>
              </HeaderRight>
            </Header>
            <Content>
              <InstrumentationRulesTable
                data={searchInput ? filterRules() : rules}
                onRowClick={handleEditRule}
                sortRules={sortRules}
              />
            </Content>
          </InstrumentationRulesContainer>
        )}
      </Container>
    </>
  );
}
