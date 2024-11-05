export const formatBytes = (bytes?: number) => {
  if (!bytes) return '0 KB/s';

  const sizes = ['Bytes', 'KB/s', 'MB/s', 'GB/s', 'TB/s'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);

  return `${value.toFixed(1)} ${sizes[i]}`;
};
