import { useState } from 'react';
import { SignalUppercase } from '@/utils';

export type ActionFormData = {
  type: string;
  name: string;
  notes: string;
  disable: boolean;
  signals: SignalUppercase[];
  details: string;
};

const INITIAL: ActionFormData = {
  type: '',
  name: '',
  notes: '',
  disable: true,
  signals: [],
  details: '',
};

export function useActionFormData() {
  const [formData, setFormData] = useState({ ...INITIAL });

  const resetFormData = () => {
    setFormData({ ...INITIAL });
  };

  const handleFormChange = (key: keyof typeof INITIAL, val: any) => {
    setFormData((prev) => ({
      ...prev,
      [key]: val,
    }));
  };

  return {
    formData,
    handleFormChange,
    resetFormData,
  };
}
