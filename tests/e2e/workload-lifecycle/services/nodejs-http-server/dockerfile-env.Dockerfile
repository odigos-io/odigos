FROM node:20.17.0-alpine
WORKDIR /app
COPY ./package.json /app
COPY ./yarn.lock /app
RUN yarn install
COPY . /app
# this test uses the NODE_OPTIONS environment variable in the Dockerfile to run a script before the main application
ENV NODE_OPTIONS="--require /app/execute_before.js"
ENV CHECK_FOR_EXECUTE_BEFORE="true"
CMD ["node", "index.js"]
