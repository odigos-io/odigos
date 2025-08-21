import os
from starlette.applications import Starlette
from starlette.responses import PlainTextResponse
from starlette.routing import Route, Mount
import logging

# Logging setup
logging.basicConfig(level=logging.INFO)
logging.getLogger().setLevel(logging.INFO)

# Sub-application
async def sub_home(request):
    return PlainTextResponse("sub hi - gunicorn + starlette")

sub_app = Starlette(routes=[
    Route("/home", sub_home)
])

# Main application
async def main_home(request):
    print(os.getgid(), flush=True)
    return PlainTextResponse("Main application - Hello world!")

app = Starlette(routes=[
    Route("/", main_home),
    Mount("/sub", app=sub_app)
])

