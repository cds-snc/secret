# Secret

Secret is a small application that allows you to share passwords, secret messages, or private links securely with another person. It does this by hiding your message behind a unique URL that can only be viewed once. Messages automatically expire after a predefined time limit if they are not viewed.

## How it works

Secret uses an AWS KMS key to [generate symmetric key pairs](https://docs.aws.amazon.com/kms/latest/developerguide/symm-asymm-concepts.html#symmetric-cmks). It uses these keys to encrypt messages and store them securely in a [DynamoDB](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Introduction.html) table. Messages can only be decrypted if the app has access to the correct KMS key in AWS. This guarantees that if the table data gets exposed, it can not be decrypted at will.

Additionally, Secret offers users the ability to encrypt the message with an additional password. The message is encrypted using [AES](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard) in CBC mode, inside the browser. This means that the app never knows anything about the additional password and even if somebody had access to both the KMS key and the table data, they could not decrypt the content.

## Technology stack 

The application leverages the [AWS Chalice framework](https://github.com/aws/chalice) to generate an easy to use API that is deployed on [AWS Lambda](https://docs.aws.amazon.com/lambda/index.html) through an API gateway. Chalice does a lot of the heavy lifting such as bundling all the dependencies and creating an appropriate [swagger.io](https://swagger.io/) definition. However, the Chalice [terraform](https://www.terraform.io/) export is strongly opinionated and therefore we only extract the parts we need and wrap it in our own terraform definition. (See `make package` inside the `Makefile`). All files related to the application itself are found inside the `/secret` directory

Additional infrastructure resources such as the AWS KMS key and the DynamoDB database are defined as infrastructure as code inside the `/aws` directory.

## Templates

The application uses HTML templates that are rendered using the [jinja2](https://jinja.palletsprojects.com/en/2.11.x/). Alternatives such as [preact](https://preactjs.com/) were considered but given the simple needs of the app, deemed too complex. However, the application can work with any framework that can render HTML and JavaScript. The JavaScript uses [jQuery](https://jquery.com/) for sake of simplicity although a full featured framework could also be used.

## Deployment

The application is deployed using [Terragrunt](https://terragrunt.gruntwork.io/) which wraps the terraform in `/aws` The terraform code contained in `/aws` is split into several independent modules that all use their own remote terraform state file. These modules know nothing about Terragrunt and are used by Terragrunt as simple infrastructure definitions.

The directory structure inside `/aws` reflects the split into independent modules. For example, `acm`, contains all the certificate logic required inside the application, while `lambda` represent the pieces needed for the application to work. The advantage is that if changes need to be made to infrastructure, should they fail, the state file has less chance of corruption and blast radius is decreased.

## Translation

The application is available in multiple languages (currently English and French). It uses the [gettext](https://www.gnu.org/software/gettext/) conventions to easily substitute language strings. The `Makefile` contains several helper aliases that should make adding translations easier. Adding a new string uses the following process:

1. A new string is introduced in a template file using the `gettext` wrapper defined in jinja. ex: `<h1>{{_("Hello")}}</h1>`.
2. Running `make babel-update` will automatically extract the string and add it to all the `.po` language files in `/secret/chalicelib/locales`. These can then be updated using a text editor.  For example inside `secret/chalicelib/locales/fr/LC_MESSAGES/base.po` you might find:
```
#: secret/chalicelib/templates/index.html:1
msgid  "Hello"
msgstr  "Bonjour"
```
3. Running `make babel-compile` will compile the `.po` files in each language into a binary `.mo` file, which the application will use to read out each string.

## To Do
- [ ] Set up CI/CD integration with GitHub actions
- [ ] Document development workflow
- [ ] Document testing workflow, hint: `make test`
- [ ] Document linting workflow, hint: `make fmt` and `make lint`

