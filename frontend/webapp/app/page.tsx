'use client';
import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useNotify, useConfig } from '@/hooks';
import { ROUTES, CONFIG, NOTIFICATION } from '@/utils';
import { FadeLoader } from '@/reuseable-components';

export default function App() {
  const router = useRouter();
  const notify = useNotify();
  const { data, error } = useConfig();

  useEffect(() => {
    router.push(ROUTES.OVERVIEW);
    // if (error) {
    //   notify({
    //     type: NOTIFICATION.ERROR,
    //     title: error.name,
    //     message: error.message,
    //   });

    // } else if (data) {
    //   const { installation } = data;

    //   switch (installation) {
    //     case CONFIG.NEW:
    //     case CONFIG.APPS_SELECTED:
    //       router.push(ROUTES.CHOOSE_SOURCES);
    //       break;
    //     case CONFIG.FINISHED:
    //       router.push(ROUTES.OVERVIEW);
    //   }
    // }
  }, []);

  return <FadeLoader />;
}
