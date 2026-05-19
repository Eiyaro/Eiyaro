package nogopow

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/zeebo/blake3"
	"golang.org/x/sync/singleflight"
)

const (
	maxCacheItems = 64
	cacheSizeMB   = 1
)

type cacheKey struct {
	headerHash Hash
	nonce      BlockNonce
}

type cacheEntry struct {
	key       cacheKey
	result    Hash
	timestamp time.Time
	size      int
}

type simpleLRU struct {
	lru   *list.List
	items map[cacheKey]*list.Element
	max   int
	lock  sync.Mutex
}

func newSimpleLRU(maxItems int) *simpleLRU {
	return &simpleLRU{
		lru:   list.New(),
		items: make(map[cacheKey]*list.Element, maxItems),
		max:   maxItems,
	}
}

func (lru *simpleLRU) Get(key cacheKey) (Hash, bool) {
	lru.lock.Lock()
	defer lru.lock.Unlock()
	
	if elem, ok := lru.items[key]; ok {
		lru.lru.MoveToFront(elem)
		return elem.Value.(*cacheEntry).result, true
	}
	
	var zero Hash
	return zero, false
}

func (lru *simpleLRU) Put(key cacheKey, value Hash) {
	lru.lock.Lock()
	defer lru.lock.Unlock()
	
	if elem, ok := lru.items[key]; ok {
		lru.lru.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.result = value
		entry.timestamp = time.Now()
		return
	}
	
	entry := &cacheEntry{
		key:       key,
		result:    value,
		timestamp: time.Now(),
		size:      32 + 32,
	}
	
	elem := lru.lru.PushFront(entry)
	lru.items[key] = elem
	
	if lru.lru.Len() > lru.max {
		oldest := lru.lru.Back()
		if oldest != nil {
			oldEntry := oldest.Value.(*cacheEntry)
			delete(lru.items, oldEntry.key)
			lru.lru.Remove(oldest)
		}
	}
}

func (lru *simpleLRU) Remove(key cacheKey) {
	lru.lock.Lock()
	defer lru.lock.Unlock()
	
	if elem, ok := lru.items[key]; ok {
		delete(lru.items, key)
		lru.lru.Remove(elem)
	}
}

func (lru *simpleLRU) Len() int {
	lru.lock.Lock()
	defer lru.lock.Unlock()
	return lru.lru.Len()
}

type Cache struct {
	lruCache *simpleLRU
	lock     sync.RWMutex
	config   *Config
	group    singleflight.Group
	memPool  sync.Pool
}

func NewCache(config *Config) *Cache {
	return &Cache{
		lruCache: newSimpleLRU(maxCacheItems),
		config:   config,
		memPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 1024)
			},
		},
	}
}

func (c *Cache) GetData(key cacheKey) (Hash, error) {
	c.lock.RLock()
	result, found := c.lruCache.Get(key)
	c.lock.RUnlock()
	
	if found {
		return result, nil
	}
	
	v, err, _ := c.group.Do(fmt.Sprintf("%x-%x", key.headerHash, key.nonce), func() (interface{}, error) {
		result, err := c.generate(key)
		if err != nil {
			return nil, err
		}
		
		c.lock.Lock()
		c.lruCache.Put(key, result)
		c.lock.Unlock()
		
		return result, nil
	})
	
	if err != nil {
		var zero Hash
		return zero, err
	}
	
	return v.(Hash), nil
}

func (c *Cache) generate(key cacheKey) (Hash, error) {
	var zero Hash
	
	if c.config == nil {
		return zero, fmt.Errorf("cache config is nil")
	}
	
	buf := c.memPool.Get().([]byte)
	defer c.memPool.Put(buf)
	
	copy(buf[:32], key.headerHash[:])
	copy(buf[32:64], key.nonce[:])
	
	hasher := blake3.New()
	hasher.Write(buf[:64])
	result := hasher.Sum(nil)
	copy(zero[:], result)
	
	return zero, nil
}

func (c *Cache) Remove(key cacheKey) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.lruCache.Remove(key)
}

type CacheStats struct {
	Size      int
	HitRate   float64
	MissRate  float64
}

func (c *Cache) Stats() CacheStats {
	c.lock.RLock()
	defer c.lock.RUnlock()
	
	size := c.lruCache.Len()
	
	return CacheStats{
		Size:     size,
		HitRate:  float64(size) / float64(maxCacheItems),
		MissRate: 1.0 - float64(size)/float64(maxCacheItems),
	}
}
