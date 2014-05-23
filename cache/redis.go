package cache

import (
	"api.umn.edu/route/util"
	"github.com/garyburd/redigo/redis"
)

func RedisConnect() (c redis.Conn, err error) {
	// Connect to Redis
	c, err = redis.Dial("tcp", util.GetenvDefault("REDIS_BIND", ":6379"))
	return
}

func RedisSet(c redis.Conn, collection string, data map[string]string) (err error) {
	// Add a hash in Redis
	_, err = c.Do("HMSET", redis.Args{}.Add(collection).AddFlat(data)...)
	if err != nil {
		panic(err)
	}

	return
}

func RedisGet(c redis.Conn, collection string) ([]string, error) {
	data, err := redis.Values(c.Do("HGETALL", collection))
	src, err := redis.Strings(data, err)

	if err != nil {
		panic(err)
	}
	return src, err
}
