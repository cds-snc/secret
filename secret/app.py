import boto3
import gettext
import jinja2
import time

from aws_lambda_powertools import Tracer
from base64 import b64encode
from chalice import BadRequestError, Chalice, NotFoundError, Response
from chalice.app import ConvertToMiddleware
from cryptography.fernet import Fernet
from os import environ
from pathlib import Path
from urllib.parse import parse_qs
from uuid import uuid4

app = Chalice(app_name="secret")

if environ.get("ENV") == "PRODUCTION":
    tracer = Tracer()
    app.register_middleware(ConvertToMiddleware(tracer.capture_lambda_handler))

_DYNAMO_CLIENT = None
_KMS_CLIENT = None

MAX_AGE_IN_DAYS = 7
MAX_SECRET_LENGTH = 64_000


def get_dynamo_client():
    global _DYNAMO_CLIENT
    if _DYNAMO_CLIENT is None:
        _DYNAMO_CLIENT = boto3.client("dynamodb", region_name="ca-central-1")
    return _DYNAMO_CLIENT


def get_kms_client():
    global _KMS_CLIENT
    if _KMS_CLIENT is None:
        _KMS_CLIENT = boto3.client("kms", region_name="ca-central-1")
    return _KMS_CLIENT


@app.route("/")
def base():
    template = render("index.html", {"lang": "en", "url_stub": "/"})
    return Response(
        template,
        status_code=200,
        headers={"Content-Type": "text/html", "Access-Control-Allow-Origin": "*"},
    )


@app.route("/{lang}")
def index(lang):
    template = render("index.html", {"lang": lang, "url_stub": "/"})
    return Response(
        template,
        status_code=200,
        headers={"Content-Type": "text/html", "Access-Control-Allow-Origin": "*"},
    )


@app.route("/decrypt/{id}")
def decrypt(id):

    dynamo_client = get_dynamo_client()
    kms_client = get_kms_client()

    response = dynamo_client.get_item(
        Key={"id": {"S": id}}, TableName=environ.get("DYNAMO_TABLE")
    )

    if "Item" in response:

        item = response["Item"]
        data_key = kms_client.decrypt(CiphertextBlob=item["key"]["B"])
        plaintext_key = b64encode(data_key["Plaintext"])
        f = Fernet(plaintext_key)
        body = f.decrypt(item["body"]["B"])

        dynamo_client.delete_item(
            Key={"id": {"S": id}}, TableName=environ.get("DYNAMO_TABLE")
        )

        return {"body": body.decode()}

    else:
        raise NotFoundError("Item is ID %s not found" % (id))


@app.route("/delete/{id}", methods=["DELETE"])
def delete(id):
    dynamo_client = get_dynamo_client()
    dynamo_client.delete_item(
        Key={"id": {"S": id}}, TableName=environ.get("DYNAMO_TABLE")
    )
    return {"status": "OK"}


@app.route("/encrypt", methods=["POST"])
def encrypt():
    body = app.current_request.json_body["body"]
    ttl = app.current_request.json_body["ttl"]
    epoch = int(time.time())

    if int(ttl) <= epoch or int(ttl) > epoch + (86_400 * MAX_AGE_IN_DAYS):
        raise BadRequestError(
            "TTL must be greater than %d and less than %d"
            % (epoch, epoch + (86_400 * MAX_AGE_IN_DAYS))
        )

    if len(body) > MAX_SECRET_LENGTH:
        raise BadRequestError(
            "Secret must be less than %d characters" % (MAX_SECRET_LENGTH)
        )

    return {"id": encrypt_and_save(body, ttl)}


@app.route("/ping")
def ping():
    return {"status": "OK"}


@app.route(
    "/slack", methods=["POST"], content_types=["application/x-www-form-urlencoded"]
)
def slack():
    data = parse_qs(app.current_request.raw_body.decode())
    if "text" in data:
        return {
            "response_type": "ephemeral",
            "blocks": [
                {
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": "Share your secret with the following link: \n>https://secret.cdssandbox.xyz/en/view/"
                        + encrypt_and_save(
                            data["text"][0], int(time.time()) + 60 * 1000
                        ),
                    },
                }
            ],
        }
    else:
        return {
            "response_type": "ephemeral",
            "text": "Sorry, an error occured generating your secret.",
        }


@app.route("/{lang}/view/{id}")
def view(lang, id):

    template = render(
        "view.html", {"lang": lang, "url_stub": "/view/" + id, "viewId": id}
    )
    return Response(
        template,
        status_code=200,
        headers={"Content-Type": "text/html", "Access-Control-Allow-Origin": "*"},
    )


def encrypt_and_save(body, ttl):
    id = str(uuid4())

    dynamo_client = get_dynamo_client()
    kms_client = get_kms_client()

    data_key = kms_client.generate_data_key(
        KeyId=environ.get("KMS_ID"), KeySpec="AES_256"
    )

    plaintext_key = b64encode(data_key["Plaintext"])
    f = Fernet(plaintext_key)

    item = {
        "id": {"S": id},
        "body": {"B": f.encrypt(body.encode("utf-8"))},
        "key": {"B": data_key["CiphertextBlob"]},
        "ttl": {"N": str(ttl)},
    }

    dynamo_client.put_item(Item=item, TableName=environ.get("DYNAMO_TABLE"))

    return id


def render(filename, context={}):
    path = Path(__file__).parent
    env = jinja2.Environment(
        loader=jinja2.FileSystemLoader(path / "./chalicelib/templates/")
    )
    context["lang"] = "en" if not context["lang"] else context["lang"]
    context["other_lang"] = "fr" if context["lang"] == "en" else "en"
    t = gettext.translation(
        "base", localedir=path / "./chalicelib/locales", languages=[context["lang"]]
    )
    t.install()
    env.globals["_"] = t.gettext
    return env.get_template(filename).render(context)
