# Odigos UI

To develop the UI, you'll need to maintain the UI libraries:

- [ui-theme](https://github.com/odigos-io/ui-theme): colors, fonts etc.
- [ui-utils](https://github.com/odigos-io/ui-utils): functions, types, consts etc.
- [ui-icons](https://github.com/odigos-io/ui-icons): SVG icons transformed to JSX elements
- [ui-components](https://github.com/odigos-io/ui-components): re-usable components (mainly from Figma -> Design System)
- [ui-containers](https://github.com/odigos-io/ui-containers): "complex components", these contain logic and are not re-usable in the same UI, these are designed to re-use across multiple deployments of the same UI (e.g. cluster, cloud, etc.)
