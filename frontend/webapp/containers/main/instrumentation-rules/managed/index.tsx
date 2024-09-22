import React, { useEffect } from 'react';
import theme from '@/styles/palette';
import { EmptyList } from '@/components';
import { useRouter } from 'next/navigation';
import { useInstrumentationRules } from '@/hooks';
import { KeyvalText, KeyvalButton, KeyvalLoader } from '@/design.system';
import {
  Header,
  Content,
  Container,
  HeaderRight,
  InstrumentationRulesContainer,
} from './styled';
import { InstrumentationRulesTable } from '@/components/overview/instrumentation-rules/rules-table';

export function ManagedInstrumentationRulesContainer() {
  const router = useRouter();

  const { isLoading, rules, refetch } = useInstrumentationRules();

  useEffect(() => {
    refetch();
  }, []);

  function handleAddRule() {
    router.push('/choose-rule');
  }

  function handleEditRule(id: string) {
    router.push(`edit-rule?id=${id}`);
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
              <div />
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
                data={rules}
                onRowClick={handleEditRule}
              />
            </Content>
          </InstrumentationRulesContainer>
        )}
      </Container>
    </>
  );
}
