# The variable TRAVIS_AWS_PROFILE is set when .travis.yml runs
#
# We prepare variables for up in UPJSON and PRODUPJSON.
# The variables coming from the AWS Parameter Store are:
# - STAGE
# - DOMAIN
# - EMAIL_FOR_NOTIFICATION_UNIT
# - PRIVATE_SUBNET_1
# - PRIVATE_SUBNET_2
# - PRIVATE_SUBNET_3
# - LAMBDA_TO_RDS_SECURITY_GROUP
# There are set as Environment variables when `aws.env` runs

stage=${STAGE}
domain=${DOMAIN}
emailForNotificationUnit=${EMAIL_FOR_NOTIFICATION_UNIT}
privateSubnet1=${PRIVATE_SUBNET_1}
privateSubnet2=${PRIVATE_SUBNET_2}
privateSubnet3=${PRIVATE_SUBNET_3}
lambdaToRdsSecurityGroup=${LAMBDA_TO_RDS_SECURITY_GROUP}

UPJSON = '.profile |= "$(TRAVIS_AWS_PROFILE)" \
		  |.stages.production |= (.domain = "unit.$(stage).$(domain)" | .zone = "$(stage).$(domain)") \
		  | .actions[0].emails |= ["$(emailForNotificationUnit)"] \
		  | .lambda.vpc.subnets |= [ "$(privateSubnet1)", "$(privateSubnet2)", "$(privateSubnet3)" ] \
		  | .lambda.vpc.security_groups |= [ "$(lambdaToRdsSecurityGroup)" ]'

PRODUPJSON = '.profile |= "$(TRAVIS_AWS_PROFILE)" \
		  |.stages.production |= (.domain = "unit.$(domain)" | .zone = "$(domain)") \
		  | .actions[0].emails |= ["$(emailForNotificationUnit)"] \
		  | .lambda.vpc.subnets |= [ "$(privateSubnet1)", "$(privateSubnet3)", "$(privateSubnet3)" ] \
		  | .lambda.vpc.security_groups |= [ "$(lambdaToRdsSecurityGroup)" ]'

# We have everything, we can run `up` now.

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
	curl -i -H "Authorization: Bearer $(call ssm,API_ACCESS_TOKEN)" https://unit.$(STAGE).$(DOMAIN)/metrics
