import React, { useEffect } from "react";
import {
  NotificationContainer,
  StyledNotification,
} from "./notification.styled";
import { KeyvalText } from "../text/text";
import CloseIcon from "@/assets/icons/X-blue.svg";
import SuccessIcon from "@/assets/icons/success-notification.svg";
import ErrorIcon from "@/assets/icons/error-notification.svg";

interface KeyvalNotificationProps {
  type: "success" | "error" | "warning" | "info";
  message: string;
  onClose?: () => void;
}

export function KeyvalNotification({
  type,
  message,
  onClose,
}: KeyvalNotificationProps) {
  useEffect(() => {
    const id = setTimeout(() => {
      onClose && onClose();
    }, 5000);

    return () => {
      clearTimeout(id);
    };
  }, []);

  function getIcon() {
    switch (type) {
      case "success":
        return <SuccessIcon />;
      case "error":
        return <ErrorIcon />;
    }
  }

  function getNotificationStyle() {
    switch (type) {
      case "error":
        return { border: "1px solid #FD3F3F" };
      default:
        return {};
    }
  }

  return (
    <NotificationContainer>
      <StyledNotification style={getNotificationStyle()}>
        {getIcon()}
        <KeyvalText weight={500} size={14}>
          {message}
        </KeyvalText>
        <CloseIcon onClick={onClose} />
      </StyledNotification>
    </NotificationContainer>
  );
}
