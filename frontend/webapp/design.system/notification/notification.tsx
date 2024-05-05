import React from "react";
import { Notification } from "@odigos-io/design-system";

interface KeyvalNotificationProps {
  type: "success" | "error" | "warning" | "info";
  message: string;
  onClose?: () => void;
}

export function KeyvalNotification(props: KeyvalNotificationProps) {
  return <Notification {...props} />;
}
