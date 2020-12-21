babel-compile:
	pybabel compile -f -D base -d secret/chalicelib/locales

babel-update:
	pybabel extract -F babel.cfg -o secret/chalicelib/locales/base.pot secret/chalicelib/templates &&\
	pybabel update -D base -i secret/chalicelib/locales/base.pot -d secret/chalicelib/locales -l en &&\
	pybabel update -D base -i secret/chalicelib/locales/base.pot -d secret/chalicelib/locales -l fr

clean:
	rm -rf secret/.chalice/deployments &&\
	rm -f aws/lambda/swagger.json &&\
	rm -f aws/lambda/deployment.zip 

deploy:
	cd terragrunt/lambda &&\
	terragrunt apply --terragrunt-non-interactive -auto-approve

fmt:
	black secret

install:
	pip3 install --user -r requirements.txt &&\
	pip3 install --user -r secret/requirements.txt

lint:
	flake8 --ignore E501,W503 secret

package: clean
	cd secret &&\
	AWS_DEFAULT_REGION=ca-central-1 chalice package --pkg-format terraform --stage prod ../aws/lambda
	jq -r '.data.template_file.chalice_api_swagger.template' aws/lambda/chalice.tf.json > aws/lambda/swagger.json
	sed -i 's/aws_lambda_function.api_handler.invoke_arn/invoke_arn/g' aws/lambda/swagger.json
	rm aws/lambda/chalice.tf.json

server:
	cd secret &&\
	chalice local

test:
	py.test -s secret/tests