package db

import (
	"sync"

	"gorm.io/gorm"
)

type Cache struct {
	mu    sync.RWMutex
	conns map[string]*gorm.DB
}

func NewCache() *Cache {
	return &Cache{conns: make(map[string]*gorm.DB)}
}

func (c *Cache) Get(dsn string) (*gorm.DB, error) {
	c.mu.RLock()
	conn, ok := c.conns[dsn]
	c.mu.RUnlock()
	if ok {
		return conn, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if conn, ok := c.conns[dsn]; ok {
		return conn, nil
	}
	conn, err := NewConnection(dsn)
	if err != nil {
		return nil, err
	}
	c.conns[dsn] = conn
	return conn, nil
}
