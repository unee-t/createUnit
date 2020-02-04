# We create a function to simplify getting variables for aws parameter store.
# The variable TRAVIS_AWS_PROFILE is set when .travis.yml runs
#
# We prepare variables for up in UPJSON and PRODUPJSON.
# The variables needed up in UPJSON and PRODUPJSON are set when `source ./aws.env` runs
# These variables can be edited in the AWS parameter store for the environment
# - STAGE
# - DOMAIN
# - EMAIL_FOR_NOTIFICATION_UNIT
# - PRIVATE_SUBNET_1
# - PRIVATE_SUBNET_2
# - PRIVATE_SUBNET_3
# - LAMBDA_TO_RDS_SECURITY_GROUP

UPJSON = '.profile |= "$(TRAVIS_AWS_PROFILE)" \
		  |.stages.production |= (.domain = "unit.$(STAGE).$(DOMAIN)" | .zone = "$(STAGE).$(DOMAIN)") \
		  | .actions[0].emails |= ["unit+$(EMAIL_FOR_NOTIFICATION_UNIT)"] \
		  | .lambda.vpc.subnets |= [ "$(PRIVATE_SUBNET_1)", "$(PRIVATE_SUBNET_2)", "$(PRIVATE_SUBNET_3)" ] \
		  | .lambda.vpc.security_groups |= [ "$(LAMBDA_TO_RDS_SECURITY_GROUP)" ]'

PRODUPJSON = '.profile |= "$(TRAVIS_AWS_PROFILE)" \
		  |.stages.production |= (.domain = "unit.$(DOMAIN)" | .zone = "$(DOMAIN)") \
		  | .actions[0].emails |= ["unit+$(EMAIL_FOR_NOTIFICATION_UNIT)"] \
		  | .lambda.vpc.subnets |= [ "$(PRIVATE_SUBNET_1)", "$(PRIVATE_SUBNET_2)", "$(PRIVATE_SUBNET_3)" ] \
		  | .lambda.vpc.security_groups |= [ "$(LAMBDA_TO_RDS_SECURITY_GROUP)" ]'
# We have everything, we can run up now.

dev:
	@echo $$AWS_ACCESS_KEY_ID
	jq $(UPJSON) up.json.in > up.json
	up deploy production

demo:
	@echo $$AWS_ACCESS_KEY_ID
	# We replace the relevant variable in the up.json file
	# We use the template defined in up.json.in for that
	jq $(UPJSON) up.json.in > up.json
	up deploy production

prod:
	@echo $$AWS_ACCESS_KEY_ID
	# We replace the relevant variable in the up.json file
	# We use the template defined in up.json.in for that
	jq $(PRODUPJSON) up.json.in > up.json
	up deploy production

test:
	curl -i -H "Authorization: Bearer $(API_ACCESS_TOKEN)" https://unit.$(STAGE).$(DOMAIN)/metrics
