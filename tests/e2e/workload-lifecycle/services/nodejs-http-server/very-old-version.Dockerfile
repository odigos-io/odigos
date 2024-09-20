FROM node:8.17.0-alpine
WORKDIR /app
COPY ./package.json /app
COPY ./yarn.lock /app
RUN yarn install
COPY . /app
CMD ["sh", "-c", "unset NODE_VERSION && node index.js"]
