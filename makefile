up:
	docker-compose up -d

init:
	docker exec -it redis-node-1 redis-cli --cluster create \
	localhost:7001 localhost:7002 localhost:7003 \
	localhost:7004 localhost:7005 localhost:7006 \
	--cluster-replicas 1 --cluster-yes

set-passwords:
	bash scripts/set-passwords.sh

down:
	docker-compose down

all: up init set-passwords