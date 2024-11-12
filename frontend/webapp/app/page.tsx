'use client';
import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
<<<<<<< HEAD
import { useConfig, useNotify } from '@/hooks';
import { ROUTES, CONFIG, NOTIFICATION } from '@/utils';
import { FadeLoader } from '@/reuseable-components';
=======
import { useNotify, useConfig } from '@/hooks';
import { FadeLoader } from '@/reuseable-components';
import { ROUTES, CONFIG, NOTIFICATION } from '@/utils';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

export default function App() {
  const router = useRouter();
  const notify = useNotify();
  const { data, error } = useConfig();

  useEffect(() => {
    if (error) {
      notify({
        type: NOTIFICATION.ERROR,
        title: error.name,
        message: error.message,
      });
<<<<<<< HEAD

      router.push(ROUTES.OVERVIEW);
    } else if (data) {
      const { installation } = data;

      switch (installation) {
        case CONFIG.NEW:
        case CONFIG.APPS_SELECTED:
          router.push(ROUTES.CHOOSE_SOURCES);
          break;
        case CONFIG.FINISHED:
          router.push(ROUTES.OVERVIEW);
      }
    }
  }, [data, error]);

=======
    } else if (data) {
      const { installation } = data;

      switch (installation) {
        case CONFIG.NEW:
        case CONFIG.APPS_SELECTED:
          router.push(ROUTES.CHOOSE_SOURCES);
          break;
        case CONFIG.FINISHED:
          router.push(ROUTES.OVERVIEW);
      }
    }
  }, [data]);

>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  return <FadeLoader />;
}
