package cache

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewACache(t *testing.T) {
	ttl := time.Second
	c := NewACache(ttl, 0)
	assert.NotNil(t, c)
	assert.Equal(t, ttl, c.ttl)
}

func TestAdd(t *testing.T) {
	c := &ACache{aCache: &aCache{m: map[ReplaceKey]*aWrapper{}}}

	k, v := ReplaceKey("foo"), ReplaceValue("bar")
	c.Add(k, v)

	fv, ok := c.aCache.m[k]
	require.True(t, ok, "%q does not exist in cache", k)
	assert.Equal(t, v, fv.v)
}

func TestGet(t *testing.T) {
	k, v := ReplaceKey("foo"), ReplaceValue("bar")
	c := &ACache{aCache: &aCache{m: map[ReplaceKey]*aWrapper{
		k: &aWrapper{v: v, expiredAt: time.Now().Add(time.Second)},
	}}}

	fv, ok := c.Get(k)
	require.True(t, ok, "%q does not exist in cache", k)
	assert.Equal(t, v, fv)
}

func TestGetNoExist(t *testing.T) {
	c := &ACache{aCache: &aCache{m: map[ReplaceKey]*aWrapper{}}}

	fv, ok := c.Get(ReplaceKey("foo"))
	assert.False(t, ok, "key should not exist in cache")
	assert.Len(t, fv, 0, "Value should be empty and is not")
}

func TestGetExpired(t *testing.T) {
	k, v := ReplaceKey("foo"), ReplaceValue("bar")
	c := &ACache{aCache: &aCache{m: map[ReplaceKey]*aWrapper{
		k: &aWrapper{v: v, expiredAt: time.Now()},
	}}}

	fv, ok := c.Get(k)
	assert.False(t, ok, "key should not exist in cache")
	assert.Len(t, fv, 0, "Value should be empty and is not")
}

func TestExpire(t *testing.T) {
	k := ReplaceKey("foo")
	c := &ACache{aCache: &aCache{m: map[ReplaceKey]*aWrapper{
		k: &aWrapper{v: ReplaceValue("bar")},
	}}}

	c.Expire(k)

	fv, ok := c.aCache.m[k]
	require.True(t, ok, "%q does not exist in cache", k)
	assert.True(t, fv.isExpired())
}

func TestCleanup(t *testing.T) {
	cleanupTime := 10 * time.Millisecond
	c := NewACache(time.Millisecond, cleanupTime)
	count := 5
	for i := 0; i < count; i++ {
		c.Add(ReplaceKey(fmt.Sprint(i)), ReplaceValue("foo"))
	}

	c.mtx.RLock()
	assert.Len(t, c.aCache.m, count)
	c.mtx.RUnlock()

	time.Sleep(cleanupTime)

	for i := 0; i < count; i++ {
		_, ok := c.Get(ReplaceKey(fmt.Sprint(i)))
		assert.False(t, ok, "key %s exists in cache and shouldn't", i)
	}
}

func TestSetFinalizer(t *testing.T) {
	cleanupTime := 10 * time.Millisecond
	c := &ACache{
		aCache: &aCache{
			m:           map[ReplaceKey]*aWrapper{},
			ttl:         0,
			cleanupTime: cleanupTime,
			stop:        make(chan struct{}),
		},
	}

	go c.cleanup()

	fin := make(chan bool, 1)
	runtime.SetFinalizer(c, func(_ *ACache) { fin <- true })
	runtime.GC()

	select {
	case <-fin:
	case <-time.After(4 * time.Second):
		t.Errorf("finalize of cache in memory didn't run")
	}
}
