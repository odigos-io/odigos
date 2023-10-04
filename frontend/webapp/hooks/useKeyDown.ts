import { useEffect } from "react";

export function useKeyDown(key, callback) {
  const handleKeyDown = (event) => {
    if (key === event.key) {
      callback(event);
    }
  };

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [key, callback]);

  return handleKeyDown;
}
