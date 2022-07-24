# Sat Apr 25 16:19:01 +0000 2015
# Mon Jan 02 15:04:05 -0700 2006
# "RubyDate"
mongorestore \
    --username "${MONGODB_ROOT_USER}" \
    --password "${MONGODB_ROOT_PASSWORD}" \
    /mnt/host/grafana-mongodb-community-plugin/integration-test/datasets/download/tweets/dump/
