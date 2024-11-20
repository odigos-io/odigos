import { NotificationNote } from '@/reuseable-components';
import styled from 'styled-components';

export const ConnectionNotification = ({ showConnectionError, destination }) => (
  <>
    {showConnectionError && (
      <NotificationNoteWrapper>
        <NotificationNote type='error' message='Connection failed. Please check your input and try again.' />
      </NotificationNoteWrapper>
    )}
    {destination?.fields && !showConnectionError && (
      <NotificationNoteWrapper>
        <NotificationNote type='default' message={`Odigos autocompleted ${destination.displayName} connection details.`} />
      </NotificationNoteWrapper>
    )}
  </>
);

const NotificationNoteWrapper = styled.div`
  margin-top: 24px;
`;
