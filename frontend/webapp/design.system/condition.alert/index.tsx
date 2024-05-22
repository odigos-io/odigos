import React from 'react';
import theme from '@/styles/palette';
import { KeyvalText } from '../text/text';
import { Error, Success } from '@/assets/icons/app';

interface Condition {
  type: string;
  status: string;
  message: string;
  last_transition_time: string;
}

interface ConditionAlertProps {
  conditions: Condition[];
}

const ConditionAlert: React.FC<ConditionAlertProps> = ({ conditions }) => {
  return (
    <div>
      <KeyvalText weight={600}>Status</KeyvalText>
      <div
        style={{
          border: `1px solid ${theme.colors.blue_grey}`,
          borderRadius: '5px',
          padding: '10px',
          marginTop: 8,
        }}
      >
        {conditions.map((condition, index) => (
          <div key={index} style={{ padding: 10 }}>
            <div style={{ display: 'flex', gap: 8 }}>
              <div
                style={{
                  width: 20,
                  height: 20,
                }}
              >
                {condition.status === 'True' ? <Success /> : <Error />}
              </div>
              <KeyvalText size={14}>{condition.message}</KeyvalText>
            </div>
            <div
              style={{
                paddingLeft: 28,
                display: 'flex',
                flexWrap: 'wrap',
                marginTop: 4,
              }}
            >
              <KeyvalText
                style={{ marginRight: 8 }}
                color={theme.text.grey}
                size={12}
              >
                {condition.type}
              </KeyvalText>
              <KeyvalText color={theme.text.grey} size={12}>
                {condition.last_transition_time}
              </KeyvalText>
            </div>
            {index !== conditions.length - 1 && (
              <div
                style={{
                  width: 2,
                  height: 30,
                  borderRadius: 20,
                  backgroundColor: theme.colors.blue_grey,
                  marginLeft: 10,
                }}
              />
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default ConditionAlert;
