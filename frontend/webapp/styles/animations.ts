import { keyframes } from 'styled-components';

type Position = 'left' | 'right' | 'top' | 'bottom' | 'center';

export const slide = {
  in: {
    left: keyframes<{ position: Position }>`
      from { transform: translateX(-100%); }
      to { transform: translateX(0); }
    `,
    right: keyframes<{ position: Position }>`
      from { transform: translateX(100%); }
      to { transform: translateX(0); }
    `,
    top: keyframes<{ position: Position }>`
      from { transform: translateY(-100%); }
      to { transform: translateY(0); }
    `,
    bottom: keyframes<{ position: Position }>`
      from { transform: translateY(100%); }
      to { transform: translateY(0); }
    `,
    center: keyframes<{ position: Position }>`
      from { transform: translate(-50%, 100%); }
      to { transform: translate(-50%, -50%); }
    `,
  },
  out: {
    left: keyframes<{ position: Position }>`
      from { transform: translateX(0); }
      to { transform: translateX(-100%); }
    `,
    right: keyframes<{ position: Position }>`
      from { transform: translateX(0); }
      to { transform: translateX(100%); }
    `,
    top: keyframes<{ position: Position }>`
      from { transform: translateY(0); }
      to { transform: translateY(-100%); }
    `,
    bottom: keyframes<{ position: Position }>`
      from { transform: translateY(0); }
      to { transform: translateY(100%); }
    `,
    center: keyframes<{ position: Position }>`
      from { transform: translate(-50%, -50%); }
      to { transform: translate(-50%, 100%); }
    `,
  },
};

export const progress = {
  in: keyframes`
    from { width: 0%; }
    to { width: 100%; }
  `,
  out: keyframes`
    from { width: 100%; }
    to { width: 0%; }
  `,
};
