// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

/*
Package cache implements a cache abstraction for use by a service.
*/
package cache

import (
	"errors"
	"sync"

	"github.com/garyburd/redigo/redis"
)

const (
	Redis = iota
)

type Cache struct {
	mutex   sync.RWMutex
	backend int
	redis_c redis.Conn
}

type KeyValuePair struct {
	key   string
	value string
}

func NewCache(backend int) (cache *Cache, err error) {
	cache = &Cache{}

	if backend == Redis {
		c, err := RedisConnect()
		cache.backend = Redis
		cache.redis_c = c
		return cache, err
	} else {
		err = errors.New("Invalid cache backend selected")
	}

	return
}

func (cache *Cache) Set(collection string, data map[string]string) (err error) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	if cache.backend == Redis {
		err = RedisSet(cache.redis_c, collection, data)
	} else {
		err = errors.New("Invalid cache backend selected")
	}

	return
}

func (cache *Cache) Get(collection string) (data []string, err error) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	if cache.backend == Redis {
		data, err = RedisGet(cache.redis_c, collection)
	} else {
		err = errors.New("Invalid cache backend selected")
	}

	return
}

func Delete(collection, key string) {
	// Method stub
}
