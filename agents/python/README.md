# @odigos/opentelemetry-python

Odigos distribution of OpenTelemetry for Python

This package is used in the odigos project to provide auto OpenTelemetry instrumentation for applications written in Python.


## Local development of `odigos-opentelemetry-python`
1. Navigate to the Project Directory:  
Open a new terminal and move to the directory containing odigos-opentelemetry-python:  
```sh
cd <ODIGOS-OPENTELEMETRY-PYTHON-PATH>
```
2. Start the Local PyPI Server:  
Build and run a local PyPI server with the following command:  
```sh
docker build -t local-pypi-server -f debug.Dockerfile . && docker run --rm --name pypi-server -p 8080:8080 local-pypi-server
```
- Note: You need to run the Docker build command each time you make changes to odigos-opentelemetry-python.  

3. Update the Development Configuration:  
In the `odigos/agents/python/setup.py` file, uncomment the DEV index-url to point to the local PyPI server.  

4. Deploy the Odiglet:  
Finally, deploy the Odiglet by running:  
```sh
make deploy-odiglet <VERSION>
```