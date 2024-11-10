import { useEffect } from 'react';

interface KeyDownOptions {
  active: boolean;
  key: Key;
  withAltKey?: boolean;
  withCtrlKey?: boolean;
  withShiftKey?: boolean;
  withMetaKey?: boolean;
}

export function useKeyDown(
  { active, key, withAltKey, withCtrlKey, withShiftKey, withMetaKey }: KeyDownOptions,
  callback: (e: KeyboardEvent) => void
) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (
        active &&
        key === e.key &&
        (!withAltKey || (withAltKey && e.altKey)) &&
        (!withCtrlKey || (withCtrlKey && e.ctrlKey)) &&
        (!withShiftKey || (withShiftKey && e.shiftKey)) &&
        (!withMetaKey || (withMetaKey && e.metaKey))
      ) {
        e.preventDefault();
        e.stopPropagation();

        callback(e);
      }
    };

    window.addEventListener('keydown', handleKeyDown);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [key, active, withAltKey, withCtrlKey, withShiftKey, withMetaKey, callback]);

  return null;
}

// Chat GPT scraped from: https://developer.mozilla.org/en-US/docs/Web/API/UI_Events/Keyboard_event_key_values
type Key =
  | 'Backspace'
  | 'Tab'
  | 'Enter'
  | 'Shift'
  | 'Control'
  | 'Alt'
  | 'Pause'
  | 'CapsLock'
  | 'Escape'
  | 'Space'
  | 'PageUp'
  | 'PageDown'
  | 'End'
  | 'Home'
  | 'ArrowLeft'
  | 'ArrowUp'
  | 'ArrowRight'
  | 'ArrowDown'
  | 'PrintScreen'
  | 'Insert'
  | 'Delete'
  | '0'
  | '1'
  | '2'
  | '3'
  | '4'
  | '5'
  | '6'
  | '7'
  | '8'
  | '9'
  | 'a'
  | 'b'
  | 'c'
  | 'd'
  | 'e'
  | 'f'
  | 'g'
  | 'h'
  | 'i'
  | 'j'
  | 'k'
  | 'l'
  | 'm'
  | 'n'
  | 'o'
  | 'p'
  | 'q'
  | 'r'
  | 's'
  | 't'
  | 'u'
  | 'v'
  | 'w'
  | 'x'
  | 'y'
  | 'z'
  | 'Meta'
  | 'ContextMenu'
  | 'F1'
  | 'F2'
  | 'F3'
  | 'F4'
  | 'F5'
  | 'F6'
  | 'F7'
  | 'F8'
  | 'F9'
  | 'F10'
  | 'F11'
  | 'F12'
  | 'NumLock'
  | 'ScrollLock'
  | 'AudioVolumeMute'
  | 'AudioVolumeDown'
  | 'AudioVolumeUp'
  | 'MediaTrackNext'
  | 'MediaTrackPrevious'
  | 'MediaStop'
  | 'MediaPlayPause'
  | 'LaunchMail'
  | 'LaunchApp1'
  | 'LaunchApp2'
  | 'Unidentified';
