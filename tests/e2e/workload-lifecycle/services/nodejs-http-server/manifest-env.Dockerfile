FROM node:20.17.0-alpine
WORKDIR /app
COPY ./package.json /app
COPY ./package-lock.json /app
RUN npm ci
COPY . /app
# Both `--require /app/execute_before.js` and `--max-old-space-size=256` are applied in manifest
ENV CHECK_FOR_APP_REQUIRE="true"
ENV CHECK_FOR_HEAP_SIZE="true"
CMD ["node", "index.js"]
