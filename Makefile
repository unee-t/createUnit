dev:
	@echo $$AWS_ACCESS_KEY_ID
	jq '.profile |= "uneet-dev" |.stages.production |= (.domain = "unit.dev.unee-t.com" | .zone = "dev.unee-t.com")| .actions[0].emails |= ["kai.hendry+unitdev@unee-t.com"]' up.json.in > up.json
	up deploy production

demo:
	@echo $$AWS_ACCESS_KEY_ID
	jq '.profile |= "uneet-demo" |.stages.production |= (.domain = "unit.demo.unee-t.com" | .zone = "demo.unee-t.com") | .actions[0].emails |= ["kai.hendry+unitdemo@unee-t.com"]' up.json.in > up.json
	up deploy production

prod:
	@echo $$AWS_ACCESS_KEY_ID
	jq '.profile |= "uneet-prod" |.stages.production |= (.domain = "unit.unee-t.com" | .zone = "unee-t.com")| .actions[0].emails |= ["kai.hendry+unitprod@unee-t.com"]' up.json.in > up.json
	up deploy production

testlocal:
	curl -i -H "Authorization: Bearer $(shell aws --profile uneet-dev ssm get-parameters --names API_ACCESS_TOKEN --with-decryption --query Parameters[0].Value --output text)" -X POST -d @tests/sample.json http://localhost:3000/create

.PHONY: dev demo prod
