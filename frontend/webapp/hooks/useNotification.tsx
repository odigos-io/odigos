import { KeyvalNotification } from "@/design.system";
import { useState } from "react";

export function useNotification() {
  const [data, show] = useState<any>(false);

  function Notification() {
    return data && <KeyvalNotification {...data} onClose={() => show(false)} />;
  }

  return { show, Notification };
}
