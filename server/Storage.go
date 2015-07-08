package solocounter

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	MAP_CAPACITY_PATH         = 1000
	MAP_CAPACITY_ADDRESS      = 1000
	CHAN_CAPACITY_REDIS_ENTRY = 10000
	REDIS_RECONNECT_INTERVAL  = 1
)

type Storage struct {
	m        *sync.Mutex
	s        map[string]*pathStore
	c        chan RedisEntry
	interval time.Duration
	window   time.Duration
	Verbose  bool
}

func NewStorage(interval, window time.Duration) *Storage {
	s := new(Storage)
	s.m = new(sync.Mutex)
	s.s = make(map[string]*pathStore, MAP_CAPACITY_PATH)
	s.interval = interval
	s.window = window
	s.c = nil
	return s
}

func (s *Storage) Simulate(numPath, perSec int) func() {
	var terminate = make(chan bool)
	for i := 0; i < numPath; i++ {
		_path := "/" + randomString()
		go func() {
			for {
				select {
				case <-time.After(time.Second / time.Duration(perSec)):
					_addr := randomString()
					s.Push(_path, _addr)
				case <-terminate:
					return
				}
			}
		}()
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			for i := 0; i < numPath; i++ {
				terminate <- true
			}
		})
	}
}

func (s *Storage) PubSub(redis_server, redis_auth, node_name string) {
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redis_server)
			if err != nil {
				return nil, err
			}
			if redis_auth != "" {
				if _, err := c.Do("AUTH", redis_auth); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	pubsub_channel := "solocounter"
	if node_name == "" {
		node_name = randomString()
	}

	go func() {
		var conn redis.Conn
		var psc redis.PubSubConn

		__connect__ := func() {
			conn = pool.Get()
			psc = redis.PubSubConn{conn}
			psc.Subscribe(pubsub_channel)
		}
		__connect__()
		defer conn.Close()

		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				if v.Channel == pubsub_channel {
					var entry RedisEntry
					err := json.Unmarshal(v.Data, &entry)
					if err != nil {
						log.Println("[error][json][unmarshal] : " + err.Error())
					} else {
						if entry.Node != node_name {
							if s.Verbose {
								log.Printf("[push][redis] %s, %s, %d", entry.Path, entry.Address, entry.Time)
							}
							s.get(entry.Path).add(entry.Address, entry.Time)
						}
					}
				}
			case redis.Subscription:
				fmt.Printf("%s: %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
				switch v.Error() {
				case fmt.Sprintf("dial tcp %s: connection refused", redis_server):
					fallthrough
				case "EOF":
					log.Println("[error][redis][subscribe] : " + v.Error() + " -> will reconnect 1 second later")
					time.Sleep(time.Second * REDIS_RECONNECT_INTERVAL)
					__connect__()
				default:
					log.Println("[error][redis][subscribe] : " + v.Error())
				}
			}
		}
	}()

	s.c = make(chan RedisEntry, CHAN_CAPACITY_REDIS_ENTRY)
	go func() {
		conn := pool.Get()
		defer conn.Close()

		for {
			select {
			case entry := <-s.c:
				entry.Node = node_name
				b, err := json.Marshal(entry)
				if err != nil {
					log.Println("[error][json][marshal] : " + err.Error())
				} else {
					_, err := conn.Do("PUBLISH", pubsub_channel, b)
					if err != nil {
						log.Println("[error][redis][publish] : " + err.Error())
					}
				}
			}
		}
	}()
}

func (s *Storage) Clean(parallel bool) {
	go func() {
		for {
			st := <-time.After(s.interval)
			s.expire(parallel)
			if s.Verbose {
				log.Printf("[clean] elasped %s", time.Now().Sub(st))
			}
		}
	}()
}

func (s *Storage) Push(path string, address string) int {
	utime := time.Now().Unix()

	if s.Verbose {
		log.Printf("[push][self] %s, %s, %d", path, address, utime)
	}

	if s.c != nil {
		s.c <- RedisEntry{Path: path, Address: address, Time: utime}
	}

	numClients := s.get(path).add(address, utime)
//
//	go func() {
//		<-time.After(s.window)
//		if s.get(path).delete(address, utime) == 0 {
//			s.m.Lock()
//			defer s.m.Unlock()
//
//			delete(s.s, path)
//		}
//	}()

	return numClients
}

func (s *Storage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	numPath := len(s.s)
	numAddr := 0
	for _, v := range s.s {
		numAddr += len(v.timeMap)
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "%d addresses in %d pathes\n\n", numAddr, numPath)
	for k, _ := range s.s {
		fmt.Fprintln(w, k)
	}
}

func (s *Storage) expire(parallel bool) {
	s.m.Lock()
	defer s.m.Unlock()

	if parallel {
		var wg sync.WaitGroup
		for k, v := range s.s {
			wg.Add(1)
			go func(_path string, _pathStore *pathStore) {
				defer wg.Done()
				if _pathStore.deleteBefore(time.Now().Add(-s.window).Unix()) == 0 {
					delete(s.s, _path)
				}
			}(k, v)
		}
		wg.Wait()
	} else {
		for k, v := range s.s {
			if v.deleteBefore(time.Now().Add(-s.window).Unix()) == 0 {
				delete(s.s, k)
			}
		}
	}
}

func (s *Storage) get(path string) *pathStore {
	s.m.Lock()
	defer s.m.Unlock()

	v, ok := s.s[path]
	if !ok {
		v = newPathStore(MAP_CAPACITY_ADDRESS)
		s.s[path] = v
	}
	return v
}

func randomString() string {
	var n uint64
	binary.Read(rand.Reader, binary.LittleEndian, &n)
	return strconv.FormatUint(n, 36)
}
