# Overview:

This codebase allows us to deploy the Unee-T APIs to add and manage users.

This codbase uses AWS Lambdas and relies on AWS Aurora's capability to call lambdas directly from Database envent (CALL and TRIGGER).

# Pre-Requisite:

- This is intended to be deployed on AWS.
- We use Travis CI for automated deployment.
- One of the dependencies for this repo is maintained on the [unee-t/env codebase](https://github.com/unee-t/env).

The following variables MUST be declared in order for this to work as intended:

## AWS variables:

These should be decleared in the AWS Parameter Store for this environment.
- STAGE
- DOMAIN
- EMAIL_FOR_NOTIFICATION_UNIT
- PRIVATE_SUBNET_1
- PRIVATE_SUBNET_2
- PRIVATE_SUBNET_3
- LAMBDA_TO_RDS_SECURITY_GROUP
- API_ACCESS_TOKEN

Make sure to check the AWS variables needed by the unee-t/env codebase in the [pre-requisite described in the README file](https://github.com/unee-t/env#pre-requisite).

## Travic CI variables:

These should be declared as Settings in Travis CI for this Repository.

### For all environments:
 - AWS_DEFAULT_REGION
 - GITHUB_TOKEN

### For dev environment:
 - AWS_ACCOUNT_USER_ID_DEV
 - AWS_ACCOUNT_SECRET_DEV
 - AWS_PROFILE_DEV

### For Demo environment:
 - AWS_ACCOUNT_USER_ID_DEMO
 - AWS_ACCOUNT_SECRET_DEMO
 - AWS_PROFILE_DEMO

### For Prod environment:
 - AWS_ACCOUNT_USER_ID_PROD
 - AWS_ACCOUNT_SECRET_PROD
 - AWS_PROFILE_PROD

# Deployment:

Deployment is done automatically with Travis CI:
- For the DEV environment: each time there is a change in the `master` repo for this codebase
- For the PROD and DEMO environment: each time we do a tag release for this repo.

# Maintenance:

To get the latest version of the go modules we need, uou can run:
`go get -u`

See the [documentation on go modules](https://blog.golang.org/using-go-modules) for more details.

# Test plan

A GET request on the root should respond "OK" if the database connection is
working. This is monitored by Postman's monitors.

There are CloudWatch alarms when functions:
* Have a high usage
* Start throwing 5xx errors
* Have a high 4xx errors
* Have a high latency

# More information:

The API is also documented in our `postman@unee-t.com` Postman account.

Caveat: AWS requires lambdas to be placed in private subnets, in order for
security groups to work.

To allow the lambda to communicate with the RDS securely you need to set the Security Groups and permissions. 
Outbound allows the lambda to talk to services inside the same "RDS" security group.
You also need a security group to allows the lambda to communicate with the outside world with wildcard 0.0.0.0/0 permissions

If you are developing from home using the Docker image, you will probably need
to whitelist your IP manually with default RDS security group if not using uneet-dev.

<img src=https://media.dev.unee-t.com/2018-09-06/my-ip.png alt="whitelist your IP address">
