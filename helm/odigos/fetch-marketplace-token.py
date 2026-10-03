#!/usr/bin/env python3
"""Retrieve a Marketplace license using the purchasing account's AWS credentials.

Requires boto3. The output file is created with mode 0600 and is never overwritten.
"""

import argparse
import json
import os
import re
import urllib.error
import urllib.request

import boto3
from botocore.auth import SigV4Auth
from botocore.awsrequest import AWSRequest


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("url", help="Odigos token URL from the Marketplace installation instructions")
    parser.add_argument("--profile", help="AWS profile in the purchasing account")
    parser.add_argument("--output", default="odigos-marketplace-token", help="new token file to create")
    args = parser.parse_args()
    if not re.fullmatch(r"https://[a-z0-9]+\.lambda-url\.us-east-1\.on\.aws/token", args.url):
        parser.error("expected the published HTTPS Lambda token URL in us-east-1")
    session = boto3.Session(profile_name=args.profile, region_name="us-east-1")
    credentials = session.get_credentials()
    if credentials is None:
        parser.error("no AWS credentials found; sign in to the purchasing account")
    request = AWSRequest(method="POST", url=args.url, data=b"")
    SigV4Auth(credentials.get_frozen_credentials(), "lambda", "us-east-1").add_auth(request)
    prepared = request.prepare()
    opener = urllib.request.build_opener(NoRedirect())
    try:
        with opener.open(urllib.request.Request(prepared.url, data=b"", headers=dict(prepared.headers), method="POST"), timeout=25) as response:
            body = json.load(response)
    except urllib.error.HTTPError as error:
        parser.exit(1, f"Token request rejected (HTTP {error.code}); check the purchase, account and invocation permissions.\n")
    except (urllib.error.URLError, TimeoutError):
        parser.exit(1, "Token service unavailable; retry later.\n")
    token = body.get("token", "")
    if not re.fullmatch(r"[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+", token):
        parser.exit(1, "Invalid token service response.\n")
    try:
        fd = os.open(args.output, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, "w") as output:
            output.write(token)
    except OSError:
        parser.exit(1, "Cannot create token file; choose a new writable --output path.\n")
    print(f"Token saved to {args.output}; expires {body.get('expires_at', 'unknown')}.")


if __name__ == "__main__":
    main()
