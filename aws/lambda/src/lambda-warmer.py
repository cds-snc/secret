from urllib import request, parse
from os import environ
from uuid import uuid4


def handler(event, context):
    req = request.Request(environ.get("BASE_URL") + "/version")
    request.urlopen(req)
    return {"statusCode": 200, "body": "OK"}
