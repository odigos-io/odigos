import TimeAgo from 'javascript-time-ago';
import en from 'javascript-time-ago/locale/en';

TimeAgo.addDefaultLocale(en);

export const useTimeAgo = () => {
  const timeAgo = new TimeAgo('en-US');

  return timeAgo;
};
