set -e

NODES=("redis-node-1" "redis-node-2" "redis-node-3" "redis-node-4" "redis-node-5" "redis-node-6")
PORTS=(7001 7002 7003 7004 7005 7006)

for i in "${!NODES[@]}"; do
    docker exec "${NODES[$i]}" redis-cli -p "${PORTS[$i]}" ACL SETUSER default on ">${REDIS_CLUSTER_PASSWORD}" "~*" "+@all"
done