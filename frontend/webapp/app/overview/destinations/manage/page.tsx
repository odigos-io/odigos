"use client";
import React, { useState } from "react";
import { NOTIFICATION, OVERVIEW } from "@/utils/constants";
import { useNotification } from "@/hooks";
import { useRouter } from "next/navigation";
import { UpdateDestinationFlow } from "@/containers/overview/destination/update.destination.flow";

export function ManageDestinationPage() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);

  const { show, Notification } = useNotification();

  const router = useRouter();

  function onSuccess(message = OVERVIEW.DESTINATION_UPDATE_SUCCESS) {
    setSelectedDestination(null);
    router.push("destinations");
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
      <UpdateDestinationFlow
        selectedDestination={selectedDestination}
        setSelectedDestination={setSelectedDestination}
        onSuccess={onSuccess}
        onError={onError}
      />
      <Notification />
    </>
  );
}
