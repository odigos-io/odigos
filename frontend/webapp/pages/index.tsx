import { useQuery } from "react-query";
import { useEffect } from "react";
import { getConfig } from "@/services/config";
import { useRouter } from "next/navigation";
import { ROUTES, CONFIG, QUERIES } from "@/utils/constants";

export default function Home() {
  const router = useRouter();
  const { isLoading, isError, isSuccess, data } = useQuery(
    [QUERIES.API_CONFIG],
    getConfig
  );

  useEffect(() => {
    router.push(ROUTES.SETUP);
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
