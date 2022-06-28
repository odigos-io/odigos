# TODO: Maybe this is pinned to specific python version due to gRPC being natively compiled, need to reasearch
FROM python:3.7 AS deps
COPY generate_requirments.py .
RUN wget https://raw.githubusercontent.com/open-telemetry/opentelemetry-python-contrib/stable/opentelemetry-instrumentation/src/opentelemetry/instrumentation/bootstrap_gen.py
RUN python generate_requirments.py
RUN mkdir /deps
RUN pip install -r requirments.txt --target /deps --upgrade

# Patch opentelemetry-python to use instrumentation for non existing packages
COPY sitecustomize.py /deps/opentelemetry/instrumentation/auto_instrumentation/sitecustomize.py

FROM ubuntu
COPY --from=deps /deps /tmp/deps
CMD ["mv","/tmp/deps/","/agent"]