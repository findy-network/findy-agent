package ssi

// Cache is keeps DIDs in memory per agent because they are so slow to load from
// wallet. Cache is not thread safe because this is not a global cache but per
// Agent.
type Cache struct {
	cache map[string]*DID
}

// Add is for the cases when DID is ready, like we know the DID`s name already.
func (c *Cache) Add(d *DID) {
	c.LazyAdd(d.Did(), d)
}

// LazyAdd is for the cases when we know the DID's name but the key is not yet
// fetched i.e. DID is launched to get key.
func (c *Cache) LazyAdd(s string, d *DID) {
	if c.cache == nil {
		c.cache = make(map[string]*DID)
	}
	old, found := c.cache[s]
	if found && old.hasKeyData() {
		return
	}
	c.cache[s] = d
}

// Get to DID by name from cache. With sure we can tell to panic if DID not
// found. That's development time use case, and normal cases the caller should
// check the return value.
func (c *Cache) Get(s string, sure bool) *DID {
	if !sure {
		v, e := c.cache[s]
		if !e {
			panic("value not exist")
		}
		return v
	}
	return c.cache[s]
}
