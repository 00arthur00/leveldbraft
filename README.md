# leveldb raft

implement a raft with backend leveldb storage.

## swagger api

``` bash
docker-compose up -d
```
load http://127.0.0.1:8901/apidocs.json from [local swagger editor](http://127.0.0.1:8080/) which is embeded in the docker-compose file.

## support operations

join cluster

memeber list

get/set/delete kv