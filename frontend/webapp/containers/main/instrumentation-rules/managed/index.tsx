import React, { useEffect, useState } from 'react';
import { useActions, useNotify } from '@/hooks';
import theme from '@/styles/palette';
import { useRouter } from 'next/navigation';
import { ACTIONS, OVERVIEW, ROUTES } from '@/utils';
import { EmptyList, ActionsTable } from '@/components';
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

export function ManagedInstrumentationRulesContainer() {
    return <>TEST</>
//   const [searchInput, setSearchInput] = useState('');

//   const router = useRouter();
//   const notify = useNotify();
//   const {
//     isLoading,
//     actions,
//     sortActions,
//     filterActionsBySignal,
//     toggleActionStatus,
//     refetch,
//   } = useActions();

//   useEffect(() => {
//     refetch();
//   }, []);

//   function handleAddAction() {
//     router.push(ROUTES.CHOOSE_ACTIONS);
//   }

//   function handleEditAction(id: string) {
//     router.push(`${ROUTES.EDIT_ACTION}?id=${id}`);
//   }

//   function filterActions() {
//     return actions.filter(
//       ({ spec: { actionName } }) =>
//         actionName &&
//         actionName.toLowerCase().includes(searchInput.toLowerCase())
//     );
//   }

//   async function onSelectStatus(ids: string[], disabled: boolean) {
//     const res = await toggleActionStatus(ids, disabled);

//     notify({
//       type: res ? 'success' : 'error',
//       message: res
//         ? OVERVIEW.ACTION_UPDATE_SUCCESS
//         : OVERVIEW.ACTION_UPDATE_ERROR,
//       title: res ? 'Success' : 'Error',
//       crdType: 'action',
//       target: '',
//     });
//   }

//   if (isLoading) return <KeyvalLoader />;

//   return (
//     <>
//       <Container>
//         {!actions?.length ? (
//           <EmptyList
//             title={OVERVIEW.EMPTY_ACTION}
//             btnTitle={OVERVIEW.ADD_NEW_ACTION}
//             btnAction={handleAddAction}
//           />
//         ) : (
//           <ActionsContainer>
//             <Header>
//               <KeyvalSearchInput
//                 containerStyle={{ padding: '6px 8px' }}
//                 placeholder={ACTIONS.SEARCH_ACTION}
//                 value={searchInput}
//                 onChange={(e) => setSearchInput(e.target.value)}
//               />
//               <HeaderRight>
//                 <KeyvalButton onClick={handleAddAction} style={{ height: 32 }}>
//                   <KeyvalText
//                     size={14}
//                     weight={600}
//                     color={theme.text.dark_button}
//                   >
//                     {OVERVIEW.ADD_NEW_ACTION}
//                   </KeyvalText>
//                 </KeyvalButton>
//               </HeaderRight>
//             </Header>
//             <Content>
//               <ActionsTable
//                 data={searchInput ? filterActions() : actions}
//                 onRowClick={handleEditAction}
//                 sortActions={sortActions}
//                 filterActionsBySignal={filterActionsBySignal}
//                 toggleActionStatus={onSelectStatus}
//               />
//             </Content>
//           </ActionsContainer>
//         )}
//       </Container>
//     </>
//   );
}
