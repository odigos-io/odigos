FROM node:20.17.0-alpine
WORKDIR /app
COPY ./package.json /app
COPY ./yarn.lock /app
RUN yarn install
COPY . /app
# this test uses the NODE_OPTIONS environment variable in the Dockerfile to run a script before the main application
ENV NODE_OPTIONS="--require /app/execute_before.js --max-old-space-size=256"
ENV CHECK_FOR_APP_REQUIRE="true"
ENV CHECK_FOR_HEAP_SIZE="true"
CMD ["node", "index.js"]
