#!/bin/sh
mkdir /tmp/otel
cd /tmp/otel
tar xvf /tmp/otel-dotnet-autoinstrumentation-0.0.1-musl.tar.gz
mv /tmp/otel/* /agent
chmod -R 777 /agent