from urllib import request, parse
from os import environ


def handler(event, context):
    data = parse.urlencode({"foo": "bar"}).encode()
    req = request.Request(environ.get("BASE_URL") + "/slack", data=data)
    req.add_header("Content-Type", "application/x-www-form-urlencoded")
    request.urlopen(req)
    return {"statusCode": 200, "body": "OK"}
