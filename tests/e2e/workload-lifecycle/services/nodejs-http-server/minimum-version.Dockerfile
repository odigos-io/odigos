FROM node:14.0.0-alpine
WORKDIR /app
COPY ./package.json /app
COPY ./package-lock.json /app
RUN npm ci
COPY . /app
CMD ["node", "index.js"]
