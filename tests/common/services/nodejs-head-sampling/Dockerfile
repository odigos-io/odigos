FROM node:22.17.0-alpine
WORKDIR /app
COPY ./package.json /app
RUN npm install --omit=dev
COPY ./src /app
CMD ["node", "index.js"]
