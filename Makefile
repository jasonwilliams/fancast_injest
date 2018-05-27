docker-build:
	docker build --tag fancast-api --build-arg SPACES_KEY=${SPACES_KEY} --build-arg SPACES_SECRET_KEY=${SPACES_SECRET_KEY} --build-arg AUTH_KEY=${AUTH_KEY} .
