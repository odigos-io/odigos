FROM node:current-alpine
WORKDIR /app
COPY ./package.json /app
COPY ./package-lock.json /app
RUN npm ci
COPY . /app
CMD ["node", "index.js"]
