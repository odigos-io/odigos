FROM node:16 AS build

WORKDIR /operator-build
COPY . .

RUN npm install

FROM busybox

COPY --from=build /operator-build/build/workspace /autoinstrumentation

RUN chmod -R go+r /autoinstrumentation
