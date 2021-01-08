from urllib import request, parse
from os import environ
from uuid import uuid4


def handler(event, context):
    data = parse.urlencode({"text": str(uuid4())}).encode()
    req = request.Request(environ.get("BASE_URL") + "/slack", data=data)
    req.add_header("Content-Type", "application/x-www-form-urlencoded")
    request.urlopen(req)
    return {"statusCode": 200, "body": "OK"}
