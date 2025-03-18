# Odigos UI

## Development

To develop the UI, you'll need to maintain the UI kit repo: [ui-kit](https://github.com/odigos-io/ui-kit)

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