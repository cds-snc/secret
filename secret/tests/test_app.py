import app
import json
import re
import time

from base64 import b64encode
from chalice.test import Client
from cryptography.fernet import Fernet
from os import environ
from secrets import token_bytes
from pytest import fixture, mark

from botocore.stub import Stubber, ANY


@fixture
def test_client():
    environ["DYNAMO_TABLE"] = "sample_table"
    environ["KMS_ID"] = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

    with Client(app.app, stage_name="testing") as client:
        yield client


@fixture
def crypto_stub():
    return {"CiphertextBlob": token_bytes(32), "Plaintext": token_bytes(32)}


@fixture
def dynamo_stub():
    client = app.get_dynamo_client()
    stub = Stubber(client)
    with stub:
        yield stub


@fixture
def kms_stub():
    client = app.get_kms_client()
    stub = Stubber(client)
    with stub:
        yield stub


def test_index(test_client):
    result = test_client.http.get("/")
    assert "<!DOCTYPE html>" in result.body.decode()
    assert result.status_code == 200


def test_index_en(test_client):
    result = test_client.http.get("/en")
    assert "Government of Canada" in result.body.decode()
    assert result.status_code == 200


def test_index_fr(test_client):
    result = test_client.http.get("/fr")
    assert "Gouvernement du Canada" in result.body.decode()
    assert result.status_code == 200


def test_ping(test_client):
    result = test_client.http.get("/ping")
    assert result.json_body == {"status": "OK"}


@mark.decrypt
def test_decrypt_with_bad_id(test_client, dynamo_stub, kms_stub):

    key = "abcd"

    dynamo_stub.add_response(
        "get_item",
        expected_params={
            "Key": {"id": {"S": key}},
            "TableName": environ["DYNAMO_TABLE"],
        },
        service_response={},
    )

    result = test_client.http.get("/decrypt/" + key)
    assert result.json_body == {
        "Code": "NotFoundError",
        "Message": "NotFoundError: Item is ID %s not found" % (key),
    }

    dynamo_stub.assert_no_pending_responses()
    kms_stub.assert_no_pending_responses()


def test_decrypt_with_good_id(test_client, dynamo_stub, kms_stub, crypto_stub):

    id = "abcd"
    body = "efgh"

    plaintext_key = b64encode(crypto_stub["Plaintext"])
    f = Fernet(plaintext_key)

    data = f.encrypt(body.encode("utf-8"))

    dynamo_stub.add_response(
        "get_item",
        expected_params={
            "Key": {"id": {"S": id}},
            "TableName": environ["DYNAMO_TABLE"],
        },
        service_response={
            "Item": {"key": {"B": crypto_stub["CiphertextBlob"]}, "body": {"B": data}}
        },
    )

    kms_stub.add_response(
        "decrypt",
        expected_params={"CiphertextBlob": crypto_stub["CiphertextBlob"]},
        service_response={"Plaintext": crypto_stub["Plaintext"]},
    )

    dynamo_stub.add_response(
        "delete_item",
        expected_params={
            "Key": {"id": {"S": id}},
            "TableName": environ["DYNAMO_TABLE"],
        },
        service_response={},
    )

    result = test_client.http.get("/decrypt/" + id)
    assert result.json_body == {"body": body}

    dynamo_stub.assert_no_pending_responses()
    kms_stub.assert_no_pending_responses()


def test_decrypt_with_utf8(test_client, dynamo_stub, kms_stub, crypto_stub):

    id = "abcd"
    body = "中文"

    plaintext_key = b64encode(crypto_stub["Plaintext"])
    f = Fernet(plaintext_key)

    data = f.encrypt(body.encode("utf-8"))

    dynamo_stub.add_response(
        "get_item",
        expected_params={
            "Key": {"id": {"S": id}},
            "TableName": environ["DYNAMO_TABLE"],
        },
        service_response={
            "Item": {"key": {"B": crypto_stub["CiphertextBlob"]}, "body": {"B": data}}
        },
    )

    kms_stub.add_response(
        "decrypt",
        expected_params={"CiphertextBlob": crypto_stub["CiphertextBlob"]},
        service_response={"Plaintext": crypto_stub["Plaintext"]},
    )

    dynamo_stub.add_response(
        "delete_item",
        expected_params={
            "Key": {"id": {"S": id}},
            "TableName": environ["DYNAMO_TABLE"],
        },
        service_response={},
    )

    result = test_client.http.get("/decrypt/" + id)
    assert result.json_body == {"body": body}

    dynamo_stub.assert_no_pending_responses()
    kms_stub.assert_no_pending_responses()


@mark.delete
def test_delete(test_client, dynamo_stub, kms_stub):
    id = "abcd"
    dynamo_stub.add_response(
        "delete_item",
        expected_params={
            "Key": {"id": {"S": id}},
            "TableName": environ["DYNAMO_TABLE"],
        },
        service_response={},
    )

    result = test_client.http.delete("/delete/" + id)
    assert result.json_body == {"status": "OK"}

    dynamo_stub.assert_no_pending_responses()
    kms_stub.assert_no_pending_responses()


@mark.encrypt
def test_encrypt_with_ttl_equal_to_epoch(test_client, dynamo_stub, kms_stub):
    epoch = int(time.time())
    payload = json.dumps({"body": "abcd", "ttl": epoch})
    result = test_client.http.post(
        "/encrypt", body=payload, headers={"Content-Type": "application/json"}
    )
    assert result.json_body == {
        "Code": "BadRequestError",
        "Message": "BadRequestError: TTL must be greater than %d and less than %d"
        % (epoch, epoch + (86_400 * app.MAX_AGE_IN_DAYS)),
    }


def test_encrypt_with_ttl_more_than_max_age(test_client, dynamo_stub, kms_stub):
    epoch = int(time.time())
    payload = json.dumps(
        {"body": "abcd", "ttl": epoch + (86_400 * app.MAX_AGE_IN_DAYS) + 1}
    )
    result = test_client.http.post(
        "/encrypt", body=payload, headers={"Content-Type": "application/json"}
    )
    assert result.json_body == {
        "Code": "BadRequestError",
        "Message": "BadRequestError: TTL must be greater than %d and less than %d"
        % (epoch, epoch + (86_400 * app.MAX_AGE_IN_DAYS)),
    }


def test_encrypt_with_body_more_than_max_length(test_client, dynamo_stub, kms_stub):
    epoch = int(time.time())
    payload = json.dumps(
        {
            "body": "0" * (app.MAX_SECRET_LENGTH + 1),
            "ttl": epoch + (86_400 * app.MAX_AGE_IN_DAYS),
        }
    )
    result = test_client.http.post(
        "/encrypt", body=payload, headers={"Content-Type": "application/json"}
    )
    assert result.json_body == {
        "Code": "BadRequestError",
        "Message": "BadRequestError: Secret must be less than %d characters"
        % (app.MAX_SECRET_LENGTH),
    }


def test_returns_an_id(test_client, dynamo_stub, kms_stub, crypto_stub):

    kms_stub.add_response(
        "generate_data_key",
        expected_params={
            "KeyId": environ["KMS_ID"],
            "KeySpec": "AES_256",
        },
        service_response={
            "CiphertextBlob": crypto_stub["CiphertextBlob"],
            "Plaintext": crypto_stub["Plaintext"],
        },
    )

    dynamo_stub.add_response(
        "put_item",
        expected_params={
            "Item": ANY,
            "TableName": environ["DYNAMO_TABLE"],
        },
        service_response={"ConsumedCapacity": {"TableName": "abcd"}},
    )

    epoch = int(time.time())
    payload = json.dumps(
        {"body": "abcd", "ttl": epoch + (86_400 * app.MAX_AGE_IN_DAYS)}
    )
    result = test_client.http.post(
        "/encrypt", body=payload, headers={"Content-Type": "application/json"}
    )
    assert "id" in result.json_body
    assert len(result.json_body["id"]) == 36

    dynamo_stub.assert_no_pending_responses()
    kms_stub.assert_no_pending_responses()

@mark.slack
def test_returns_an_id_to_slack(test_client, dynamo_stub, kms_stub, crypto_stub):

    kms_stub.add_response(
        "generate_data_key",
        expected_params={
            "KeyId": environ["KMS_ID"],
            "KeySpec": "AES_256",
        },
        service_response={
            "CiphertextBlob": crypto_stub["CiphertextBlob"],
            "Plaintext": crypto_stub["Plaintext"],
        },
    )

    dynamo_stub.add_response(
        "put_item",
        expected_params={
            "Item": ANY,
            "TableName": environ["DYNAMO_TABLE"],
        },
        service_response={"ConsumedCapacity": {"TableName": "abcd"}},
    )

    payload = "text=foo"
    result = test_client.http.post(
        "/slack", body=payload, headers={"Content-Type": "application/x-www-form-urlencoded"}
    )
    assert "response_type" in result.json_body
    assert result.json_body["response_type"] == "ephemeral"

    assert "blocks" in result.json_body
    assert re.search('view\\/.{36}$', result.json_body["blocks"][0]["text"]["text"])

    dynamo_stub.assert_no_pending_responses()
    kms_stub.assert_no_pending_responses()

def test_returns_an_error_to_slack(test_client):
    payload = ""
    result = test_client.http.post(
        "/slack", body=payload, headers={"Content-Type": "application/x-www-form-urlencoded"}
    )
    assert "response_type" in result.json_body
    assert result.json_body["response_type"] == "ephemeral"

    assert "text" in result.json_body
    assert result.json_body["text"] == "Sorry, an error occured generating your secret."