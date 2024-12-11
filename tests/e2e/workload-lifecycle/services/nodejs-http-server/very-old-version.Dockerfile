FROM node:8.17.0-alpine
WORKDIR /app
COPY ./package.json /app
COPY ./package-lock.json /app
RUN npm ci
COPY . /app
ENV NODE_VERSION=""
CMD ["node", "index.js"]
