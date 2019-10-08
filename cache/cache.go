package cache

import (
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env"
	"github.com/gomodule/redigo/redis"
)

type config struct {
	Server      string        `env:"REDIS_SERVER" envDefault:"127.0.0.1:6379"`
	MaxIdle     int           `env:"REDIS_MAX_IDLE" envDefault:"10"`
	MaxActive   int           `env:"REDIS_MAX_ACTIVE" envDefault:"100"`
	IdleTimeout time.Duration `env:"REDIS_IDLE_TIMEOUT" envDefault:"24s"`
	Wait        bool          `env:"REDIS_WAIT" envDefault:"true"`
}

/*CachePool maintains a pool of connections.The application calls the Get method to get
a connection from the pool and the connection's Close method to return the
connection's resources to the pool.*/
var CachePool *redis.Pool

func init() {
	cfg := config{}

	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
	}

	CachePool = &redis.Pool{
		MaxIdle:     cfg.MaxIdle,
		IdleTimeout: cfg.IdleTimeout,
		MaxActive:   cfg.MaxActive,
		Wait:        cfg.Wait,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.Server)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}
			return c, nil
		},
	}
}

//SetEx sets a new key in to database and also expiry time in ms
func SetEx(RConn *redis.Conn, key string, ttl int, data interface{}) (interface{}, error) {
	return (*RConn).Do("SETEX", key, ttl, data)
}

//GetString returns the value for the key in string format
func GetString(RConn *redis.Conn, key string) (string, error) {
	return redis.String((*RConn).Do("GET", key))
}

//GetInt returns value of key in integer format
func GetInt(RConn *redis.Conn, key string) (int, error) {
	return redis.Int((*RConn).Do("GET", key))
}

// Exists checks whether a key is present in the database
func Exists(RConn *redis.Conn, key string) (bool, error) {
	count, err := redis.Int((*RConn).Do("EXISTS", key))
	if count == 0 {
		return false, err
	}
	return true, err

}

//DeleteAllKeys deletes all keys from the current database
func DeleteAllKeys(RConn *redis.Conn) error {
	_, err := (*RConn).Do("FLUSHDB")
	return err

}

// DeleteKey deletes a key given the key name
func DeleteKey(RConn *redis.Conn, key string) (int, error) {
	return redis.Int((*RConn).Do("UNLINK", key))
}

// DeleteKeysByPattern delete all keys matching a pattern
func DeleteKeysByPattern(RConn *redis.Conn, pattern string) {
	start := time.Now()
	var (
		cursor        int64
		items         []string
		noOfKeysToDel int64
	)

	for {
		values, err := redis.Values((*RConn).Do("SCAN", cursor, "MATCH", pattern, "COUNT", 100))
		if err != nil {
			fmt.Println("cache:", err)
			return
		}

		_, err = redis.Scan(values, &cursor, &items)
		if err != nil {
			continue
		}

		for _, item := range items {
			ttl, _ := redis.Int((*RConn).Do("TTL", item))
			if time.Now().Add(-time.Duration(600-ttl) * time.Second).Before(start) {
				err = (*RConn).Send("UNLINK", item)
				if err != nil {
					log.Println("cache:", err)
				}
				noOfKeysToDel++
			}

		}

		if cursor == 0 {
			break
		}

	}

}
