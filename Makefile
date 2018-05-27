docker-build:
	docker build --tag jayflux/fancast-api --build-arg DB_USER=${DB_USER} --build-arg DB_NAME=${DB_NAME} --build-arg DB_PASS=${DB_PASS} --build-arg SPACES_KEY=${SPACES_KEY} --build-arg SPACES_SECRET_KEY=${SPACES_SECRET_KEY} --build-arg AUTH_KEY="${AUTH_KEY}" .
