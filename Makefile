PROFILE := ins-dev
define ssm
$(shell aws --profile $(PROFILE) ssm get-parameters --names $1 --with-decryption --query Parameters[0].Value --output text)
endef

DEVUPJSON = '.profile |= "$(PROFILE)" \
		  |.stages.production |= (.domain = "unit.$(call ssm,STAGE).$(call ssm,DOMAIN)" | .zone = "$(call ssm,STAGE).$(call ssm,DOMAIN)") \
		  | .actions[0].emails |= ["unit+$(call ssm,EMAIL_FOR_NOTIFICATION_GENERIC)"] \
		  | .lambda.vpc.subnets |= [ "$(call ssm,PRIVATE_SUBNET_1)", "$(call ssm,PRIVATE_SUBNET_2)", "$(call ssm,PRIVATE_SUBNET_3)" ] \
		  | .lambda.vpc.security_groups |= [ "$(call ssm,DEFAULT_SECURITY_GROUP)" ]'


dev:
	@echo $$AWS_ACCESS_KEY_ID
	jq $(DEVUPJSON) up.json.in > up.json
	up deploy production

test:
	curl -i -H "Authorization: Bearer $(call ssm,API_ACCESS_TOKEN)" https://unit.$(call ssm,STAGE).$(call ssm,DOMAIN)/metrics
