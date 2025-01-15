import React, { useEffect } from 'react';
import { useDarkModeStore } from '@/store';
import { IconButton } from '@/reuseable-components';
import { LightOffIcon, LightOnIcon } from '@/assets';

interface Props {}

export const ToggleDarkMode: React.FC<Props> = () => {
  const { darkMode, setDarkMode } = useDarkModeStore();

  useEffect(() => {
    const lsValue = localStorage.getItem('darkMode');
    if (!!lsValue) setDarkMode(lsValue == 'true');
  }, []);

  const handleToggle = () => {
    setDarkMode(!darkMode);
    localStorage.setItem('darkMode', JSON.stringify(!darkMode));
  };

  return (
    <IconButton onClick={handleToggle} tooltip={darkMode ? 'Light Mode' : 'Dark Mode'}>
      {darkMode ? <LightOffIcon size={18} /> : <LightOnIcon size={18} />}
    </IconButton>
  );
};
