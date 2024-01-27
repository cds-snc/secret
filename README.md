# Encrypted Message Crypté

Encrypted Message Crypté is a small application that allows you to share passwords, secret messages, or private links securely with another person. It does this by hiding your message behind a unique URL that can only be viewed once. Messages automatically expire after a predefined time limit if they are not viewed.

Below is a short animated GIF showing the application in action.

![encrypted message](https://github.com/cds-snc/secret/assets/867334/60f1588a-f7a4-47d2-8cc7-ee356d6c5b89)

## How does it work?

The application uses a combination of AES-256 encryption and a unique URL to hide your message. When you enter a message, the application generates a unique URL that contains a reference ID to decrypt your message. You can then share this URL with the intended recipient. When the recipient opens the URL, the application will decrypt the message and display it to them once they click the "View secret" button (this avoids accidental views through URL unfurling). The message is then deleted from the database and can no longer be viewed.

Optionally, you can also encrypt the message on the client side before sending it to the server. This means that the server never sees the unencrypted message. This is useful if you don't trust the server or if you want to add an extra layer of security. However, the recipient will need to know the password you used to encrypt the message in order to decrypt it.

## Backends

The application allows you to specify both an encryption backend as well as a storage backend. This allows you to run the application across multiple cloud service providers or even on your own hardware. The interfaces for the encryption and storage backends are defined in the `encryption/encryption_backend.go` and `storage/storage_backend.go` files respectively.

The application comes with two encryption backends out of the box:

* `encryption/rsa.go` - This backend uses RSA encryption to encrypt the message. Please use the `make generate-keys` command before running the application to generate the necessary keys and not use the default ones.
* `encryption/aws_kms.go` - This backend uses the AWS Key Management Service (KMS) to encrypt the message.

The application also comes with two storage backends out of the box:

* `storage/dynamodb.go` - This backend uses DynamoDB to store the encrypted message.
* `storage/in_memory.go` - This backend uses an in-memory map to store the encrypted message. This backend is useful for testing and will not persist data across restarts.

If you would like to build your own binaries with custom backends, take a look at the `cmd/app` and `cmd/lambda_app` directories for inspiration.

## License

This project is released under the terms of the MIT license. See [LICENSE](LICENSE) for more information or see https://opensource.org/licenses/MIT.