package beeorm

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
)

const requestCacheKey = "_request"

type LocalCachePoolConfig interface {
	GetCode() string
	GetLimit() int
}

type localCachePoolConfig struct {
	code  string
	limit int
	lru   *lru.Cache
	m     sync.Mutex
}

func (p *localCachePoolConfig) GetCode() string {
	return p.code
}

func (p *localCachePoolConfig) GetLimit() int {
	return p.limit
}

type LocalCache struct {
	engine *Engine
	config *localCachePoolConfig
}

type ttlValue struct {
	value interface{}
	time  int64
}

func (c *LocalCache) GetPoolConfig() LocalCachePoolConfig {
	return c.config
}

func (c *LocalCache) GetSet(key string, ttl time.Duration, provider func() interface{}) interface{} {
	val, has := c.Get(key)
	if has {
		ttlVal := val.(ttlValue)
		seconds := int64(ttl.Seconds())
		if seconds == 0 || time.Now().Unix()-ttlVal.time <= seconds {
			return ttlVal.value
		}
	}
	userVal := provider()
	val = ttlValue{value: userVal, time: time.Now().Unix()}
	c.Set(key, val)
	return userVal
}

func (c *LocalCache) Get(key string) (value interface{}, ok bool) {
	c.config.m.Lock()
	defer c.config.m.Unlock()

	value, ok = c.config.lru.Get(key)
	if c.engine.hasLocalCacheLogger {
		c.fillLogFields("GET", "GET "+key, !ok)
	}
	return
}

func (c *LocalCache) MGet(keys ...string) []interface{} {
	c.config.m.Lock()
	defer c.config.m.Unlock()

	results := make([]interface{}, len(keys))
	misses := 0
	for i, key := range keys {
		value, ok := c.config.lru.Get(key)
		if !ok {
			misses++
			value = nil
		}
		results[i] = value
	}
	if c.engine.hasLocalCacheLogger {
		c.fillLogFields("MGET", "MGET "+strings.Join(keys, " "), misses > 0)
	}
	return results
}

func (c *LocalCache) Set(key string, value interface{}) {
	c.config.m.Lock()
	defer c.config.m.Unlock()
	c.config.lru.Add(key, value)
	if c.engine.hasLocalCacheLogger {
		c.fillLogFields("SET", fmt.Sprintf("SET %s %v", key, value), false)
	}
}

func (c *LocalCache) MSet(pairs ...interface{}) {
	max := len(pairs)
	c.config.m.Lock()
	defer c.config.m.Unlock()
	for i := 0; i < max; i += 2 {
		c.config.lru.Add(pairs[i], pairs[i+1])
	}
	if c.engine.hasLocalCacheLogger {
		message := "MSET "
		for _, v := range pairs {
			message += fmt.Sprintf(" %v", v)
		}
		c.fillLogFields("MSET", message, false)
	}
}

func (c *LocalCache) HMGet(key string, fields ...string) map[string]interface{} {
	c.config.m.Lock()
	defer c.config.m.Unlock()

	l := len(fields)
	results := make(map[string]interface{}, l)
	value, ok := c.config.lru.Get(key)
	misses := 0
	for _, field := range fields {
		if !ok {
			results[field] = nil
			misses++
		} else {
			val, has := value.(map[string]interface{})[field]
			if !has {
				results[field] = nil
				misses++
			} else {
				results[field] = val
			}
		}
	}
	if c.engine.hasLocalCacheLogger {
		c.fillLogFields("HMGET", "HMGET "+key+" "+strings.Join(fields, " "), misses > 0)
	}
	return results
}

func (c *LocalCache) HMSet(key string, fields map[string]interface{}) {
	c.config.m.Lock()
	defer c.config.m.Unlock()

	m, has := c.config.lru.Get(key)
	if !has {
		m = make(map[string]interface{})
		c.config.lru.Add(key, m)
	}
	for k, v := range fields {
		m.(map[string]interface{})[k] = v
	}
	if c.engine.hasLocalCacheLogger {
		message := "HMSET " + key + " "
		for k, v := range fields {
			message += fmt.Sprintf(" %s %v", k, v)
		}
		c.fillLogFields("HMSET", message, false)
	}
}

func (c *LocalCache) Remove(keys ...string) {
	c.config.m.Lock()
	defer c.config.m.Unlock()
	for _, v := range keys {
		c.config.lru.Remove(v)
	}
	if c.engine.hasLocalCacheLogger {
		c.fillLogFields("REMOVE", "REMOVE "+strings.Join(keys, " "), false)
	}
}

func (c *LocalCache) Clear() {
	c.config.m.Lock()
	defer c.config.m.Unlock()
	c.config.lru.Clear()
	if c.engine.hasLocalCacheLogger {
		c.fillLogFields("CLEAR", "CLEAR", false)
	}
}

func (c *LocalCache) GetObjectsCount() int {
	c.config.m.Lock()
	defer c.config.m.Unlock()
	return c.config.lru.Len()
}

func (c *LocalCache) fillLogFields(operation, query string, cacheMiss bool) {
	fillLogFields(c.engine.queryLoggersLocalCache, c.config.GetCode(), sourceLocalCache, operation, query, nil, cacheMiss, nil)
}
