import { useState } from 'react';

export const useGenericForm = <Form = Record<string, any>>(initialFormData: Form) => {
  const [formData, setFormData] = useState<Form>({ ...initialFormData });
  const [formErrors, setFormErrors] = useState<Partial<Record<keyof Form, string>>>({});

  const handleFormChange = (key?: keyof typeof formData, val?: any, obj?: typeof formData) => {
    if (!!key) {
      // this is for cases where the form contains objects such as "exportedSignals",
      // the object's child is targeted with a ".dot" for example: "exportedSignals.logs"

      const [parentKey, childKey] = (key as string).split('.');

      if (!!childKey) {
        setFormData((prev) => ({ ...prev, [parentKey]: { ...prev[parentKey], [childKey]: val } }));
      } else {
        setFormData((prev) => ({ ...prev, [parentKey]: val }));
      }
    } else if (!!obj) {
      setFormData({ ...obj });
    }
  };

  const handleErrorChange = (key?: keyof typeof formErrors, val?: string, obj?: typeof formErrors) => {
    if (!!key) {
      setFormErrors((prev) => ({ ...prev, [key]: val }));
    } else if (!!obj) {
      setFormErrors({ ...obj });
    }
  };

  const resetFormData = () => {
    setFormData({ ...initialFormData });
    setFormErrors({});
  };

  return {
    formData,
    formErrors,
    handleFormChange,
    handleErrorChange,
    resetFormData,
  };
};
