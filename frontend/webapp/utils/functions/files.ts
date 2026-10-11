export const downloadFileFromURL = async (url: string) => {
  // we create a hidden a-tag instead of window.open to avoid browser blocking popups
  const link = document.createElement('a');
  link.href = url;
  link.download = `odigos-diagnose-${new Date().getTime()}.tar.gz`;
  document.body.appendChild(link);

  // trigger the download
  link.click();

  // cleanup
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
};
