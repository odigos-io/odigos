import { useState } from 'react';

export const useGenericForm = <Form = Record<string, any>>(initialFormData: Form) => {
  function copyInitial() {
    // this is to avoid reference issues with the initial form data,
    // when an object has arrays or objects as part of it's values, a simple spread operator won't work, the children would act as references,
    // so we use JSON.parse(JSON.stringify()) to create a deep copy of the object without affecting the original
    return JSON.parse(JSON.stringify(initialFormData));
  }

  const [formData, setFormData] = useState<Form>(copyInitial());
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
    setFormData(copyInitial());
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
