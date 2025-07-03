start-redis-nods:
	docker-compose up -d redis-node-1 redis-node-2 redis-node-3 redis-node-4 redis-node-5 redis-node-6

init-redis-cluster:
	#docker exec -it redis-node-1 redis-cli --cluster create \
#	localhost:7001 localhost:7002 localhost:7003 \
#	localhost:7004 localhost:7005 localhost:7006 \
#	--cluster-replicas 1 --cluster-yes
	docker exec -it redis-node-1 redis-cli --cluster create \
	redis-node-1:7001 redis-node-2:7002 redis-node-3:7003 \
	redis-node-4:7004 redis-node-5:7005 redis-node-6:7006 \
	--cluster-replicas 1 --cluster-yes

include ./config/config.env
export REDIS_CLUSTER_PASSWORD

set-cluster-passwords:
	bash scripts/set-cluster-passwords.sh

down:
	docker-compose down

start-monitoring:
	docker-compose up -d grafana prometheus

start-app:
	docker-compose up --build -d notification-service

all: start-redis-nods init-redis-cluster set-cluster-passwords start-monitoring start-app