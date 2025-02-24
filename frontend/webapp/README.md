# Odigos UI

## Development

To develop the UI, you'll need to maintain the UI libraries:

- [ui-theme](https://github.com/odigos-io/ui-theme): colors, fonts etc.
- [ui-utils](https://github.com/odigos-io/ui-utils): functions, types, consts etc.
- [ui-icons](https://github.com/odigos-io/ui-icons): SVG icons transformed to JSX elements
- [ui-components](https://github.com/odigos-io/ui-components): re-usable components (mainly from Figma -> Design System)
- [ui-containers](https://github.com/odigos-io/ui-containers): "complex components", these contain logic and are not re-usable in the same UI, these are designed to re-use across multiple deployments of the same UI (e.g. cluster, cloud, etc.)

## Running Locally

1. Install dependencies:
    ```bash
    yarn install
    ```

2. Create a client build:
    ```bash
    yarn build
    ```

3. Create a server build:
    ```bash
    yarn back:build
    ```

4. Start the server:
    ```bash
    yarn back:start
    ```
    You should now be able to visit the UI on [localhost:8085](http://localhost:8085).

5. Note: if you want to get real-time code updates, you'll have to run the client seperately:
    ```bash
    yarn dev
    ```
    You should now be able to visit the UI on [localhost:3000](http://localhost:3000).