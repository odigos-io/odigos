import { useRouter, useSearchParams } from 'next/navigation';
import { ROUTES, SKIP_TO_SUMMERY_QUERY_PARAM } from '@/utils';

export const useSetupHelpers = () => {
  const router = useRouter();
  const params = useSearchParams();

  const isSkipToSummary = !!params.get(SKIP_TO_SUMMERY_QUERY_PARAM);
  const skipToSummaryQuerystring = isSkipToSummary ? `?${SKIP_TO_SUMMERY_QUERY_PARAM}=true` : '';

  // If we do not want to show the "go to summary button" in setup pages, we have to pass "undefined" as prop
  const onClickSummary = isSkipToSummary ? () => router.push(ROUTES.SETUP_SUMMARY) : undefined;
  const onClickRouteFromSummary = (path: string) => router.push(path + skipToSummaryQuerystring);

  return { onClickSummary, onClickRouteFromSummary };
};
