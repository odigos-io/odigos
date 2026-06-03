'use client';

import React from 'react';
import type { ProfileType } from '@/hooks';

interface ProfileTypeToggleProps {
  value: ProfileType;
  onChange: (profileType: ProfileType) => void;
}

const OPTIONS: { label: string; value: ProfileType }[] = [
  { label: 'CPU', value: 'cpu' },
  { label: 'Memory', value: 'alloc_space' },
];

// ProfileTypeToggle is a minimal CPU⇄memory switch for the profiling view.
// Selecting an option updates the active profile type, which re-queries the backend flame graph.
export const ProfileTypeToggle: React.FC<ProfileTypeToggleProps> = ({ value, onChange }) => {
  return (
    <div role='group' aria-label='Profile type' style={{ display: 'inline-flex', gap: 4 }}>
      {OPTIONS.map((opt) => {
        const selected = opt.value === value;
        return (
          <button
            key={opt.value}
            type='button'
            aria-pressed={selected}
            onClick={() => {
              if (!selected) onChange(opt.value);
            }}
            style={{
              padding: '4px 12px',
              borderRadius: 6,
              cursor: 'pointer',
              border: '1px solid currentColor',
              opacity: selected ? 1 : 0.6,
              fontWeight: selected ? 600 : 400,
            }}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
};
