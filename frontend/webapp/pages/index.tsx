import { useQuery } from "react-query";
import { useEffect } from "react";
import { getConfig } from "@/services/config";
import { CONFIG } from "@/utils/constants/config";
import { useRouter } from "next/router";
import { ROUTES } from "@/utils/constants/routes";

export default function Home() {
  const router = useRouter();
  const { isLoading, isError, isSuccess, data } = useQuery(
    ["apiConfig"],
    getConfig
  );

  useEffect(() => {
    data && renderCurrentPage();
  }, [data]);

  function renderCurrentPage() {
    const { installation } = data;
    switch (installation) {
      case CONFIG.FINISHED:
        router.push(ROUTES.SETUP);
    }
  }
}
