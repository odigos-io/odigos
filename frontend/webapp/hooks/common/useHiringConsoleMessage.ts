import { useEffect } from 'react';

export default function useHiringConsoleMessage() {
  useEffect(() => {
    // ASCII art Odigos logo header
    console.log(
      `%c\n#######  ########  ####  ######    #######   ######  \n##     ## ##     ##  ##  ##    ##  ##     ## ##    ## \n##     ## ##     ##  ##  ##        ##     ## ##       \n##     ## ##     ##  ##  ##   #### ##     ##  ######  \n##     ## ##     ##  ##  ##    ##  ##     ##       ## \n##     ## ##     ##  ##  ##    ##  ##     ## ##    ## \n #######  ########  ####  ######    #######   ######  \n`,
      'color: #000000; font-family: monospace; font-size: 10px; font-weight: bold; line-height: 1;',
    );
    // Dev-style message
    const divider = '%c──────────────────────────────────────────────────────────────';
    const dividerStyle = 'color: #888; font-family: monospace; font-size: 12px;';
    console.log(divider, dividerStyle);
    console.log('%c🚀 WE ARE HIRING! 🚀', 'color: #ff6b35; font-family: monospace; font-size: 18px; font-weight: bold;');
    console.log(divider, dividerStyle);
    console.log('%cJoin the Odigos team and help us build the future of observability.', 'color: #007bff; font-family: monospace; font-size: 14px;');
    console.log('%cWe are looking for talented developers who love open source and cloud-native tech.', 'color: #28a745; font-family: monospace; font-size: 13px;');
    console.log(divider, dividerStyle);
    console.log('%cOpen positions & details:', 'color: #17a2b8; font-family: monospace; font-size: 13px; font-weight: bold;');
    console.log('%chttps://www.comeet.com/jobs/odigos/5A.001', 'color: #007bff; font-family: monospace; font-size: 13px; text-decoration: underline;');
    console.log(divider, dividerStyle);
    console.log('%cTip: Reach out if you have questions or want to contribute!', 'color: #6f42c1; font-family: monospace; font-size: 12px; font-style: italic;');
    console.log(divider, dividerStyle);
  }, []);
}
