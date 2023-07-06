"use client";
import { useQuery } from "react-query";
import { useEffect } from "react";
import { getConfig } from "@/services/config";
import { CONFIG } from "@/utils/constants/config";
import { useRouter } from "next/navigation";
import { ROUTES } from "@/utils/constants/routes";

export default function Home() {
  const router = useRouter();
  const { isLoading, isError, isSuccess, data } = useQuery(
    ["apiConfig"],
    getConfig
  );

  useEffect(() => {
    console.log({ router });
    router.push(ROUTES.SETUP);
    // data && renderCurrentPage();
  }, [data]);

  function renderCurrentPage() {
    const { installation } = data;
    // console.log({ installation });

    switch (installation) {
      case CONFIG.FINISHED:
        router.push(ROUTES.SETUP);
    }
  }
}
