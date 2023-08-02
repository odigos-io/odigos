"use client";
import React, { useEffect } from "react";
import { NOTIFICATION, OVERVIEW } from "@/utils/constants";
import { useNotification } from "@/hooks";
import { NewDestinationFlow } from "@/containers/overview/destination/new.destination.flow";
import { useRouter } from "next/navigation";

export default function CreateDestinationPage() {
  const { show, Notification } = useNotification();
  const router = useRouter();

  function onSuccess(message = OVERVIEW.DESTINATION_UPDATE_SUCCESS) {
    show({
      type: NOTIFICATION.SUCCESS,
      message,
    });
  }

  function onError({ response }) {
    const message = response?.data?.message;
    show({
      type: NOTIFICATION.ERROR,
      message,
    });
  }

  return (
    <>
      <NewDestinationFlow onSuccess={onSuccess} onError={onError} />
      <Notification />
    </>
  );
}
