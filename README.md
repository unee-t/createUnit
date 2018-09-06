Please view documentation in Postman under the folder Unit.

# Test plan

A GET request on the root should respond "OK" if the database connection is
working. This is monitored by Postman's monitors.

There are CloudWatch alarms when functions:
* Have a high usage
* Start throwing 5xx errors
* Have a high 4xx errors
* Have a high latency

# Timeout connecting to the database?

The RDS database is protected by a password and a "CIDR whitelist", implemented
by security groups configuration. Lambdas are white listed by putting them into TWO
security groups, for example on the dev account:
* sg-0b83472a34bc17400 "RDS", applied to RDS & Lambda
* sg-0f4dadb564041855b "secondary" only on Lambda

sg-0b83472a34bc17400 allows inbound 3306 from sg-0f4dadb564041855b, allowing
the lambda to communicate with the RDS securely. Outbound allows it talk to
services inside the same "RDS" security group.

sg-0f4dadb564041855b allows the lambda to communicate with the outside world
with wildcard 0.0.0.0/0 permissions

Caveat: AWS requires lambdas to be placed in private subnets, in order for security groups to work.

If you are developing from home using the Docker image, you will probably need
to whitelist your IP manually with default RDS security group. Notice this
differs account to account.

<img src=https://media.dev.unee-t.com/2018-09-06/my-ip.png alt="whitelist your IP address">
