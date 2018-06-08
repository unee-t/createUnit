curl -X POST  \
	-H "Authorization: Bearer $(aws --profile uneet-dev ssm get-parameters --names API_ACCESS_TOKEN --with-decryption --query Parameters[0].Value --output text)" \
	http://localhost:3000/create
