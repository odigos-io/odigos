# To build one auto-instrumentation image for Python, please:
# - Ensure the packages are installed in the `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#   one init container will be created to copy all the content in `/autoinstrumentation` directory to your app's container. Then
#   update the `PYTHONPATH` environment variable accordingly. To achieve this, you can mimic the one in `autoinstrumentation/python/Dockerfile`
#   by using multi-stage builds. In the first stage, install all the required packages in one custom directory with `pip install --target`.
#   Then in the second stage, copy the directory to `/autoinstrumentation`.
# - Ensure you have `opentelemetry-distro` and `opentelemetry-instrumentation` or your customized alternatives installed.
#   Those two packages are essential to Python auto-instrumentation.
# - Grant the necessary access to `/autoinstrumentation` directory. `chmod -R go+r /autoinstrumentation`

FROM python:3.10-alpine AS build

WORKDIR /operator-build

ADD requirements.txt .

RUN mkdir workspace && pip install --target workspace -r requirements.txt

FROM busybox

COPY --from=build /operator-build/workspace /autoinstrumentation

RUN chmod -R go+r /autoinstrumentation