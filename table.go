package main

import "bytes"

// rules say at most 10k unique stations, keep the load factor low
const TABLE_BITS = 16
const TABLE_SIZE = 1 << TABLE_BITS
const TABLE_MASK = TABLE_SIZE - 1

const FNV_OFFSET = 14695981039346656037
const FNV_PRIME = 1099511628211

type entry struct {
  name []byte // points into the mmapped file, no copy
  hash uint64
  Stats
}

// open addressing + linear probing, replaces map[string]*Stats in the hot
// loop (mapaccess2_faststr + memhash was ~20% of the profile)
type table struct {
  entries []entry
  used    []int // indices in use, so we don't scan 64k slots at the end
}

func new_table() *table {
  return &table{entries: make([]entry, TABLE_SIZE)}
}

func (t *table) get(name []byte, h uint64) *Stats {
  // top bits: with word at a time hashing the low bits of h only depend
  // on the first couple of bytes of each word
  i := h >> (64 - TABLE_BITS)
  for {
    e := &t.entries[i]
    if e.name == nil {
      e.name = name
      e.hash = h
      e.min = 999
      e.max = -999
      t.used = append(t.used, int(i))
      return &e.Stats
    }
    if e.hash == h && bytes.Equal(e.name, name) {
      return &e.Stats
    }
    i = (i + 1) & TABLE_MASK
  }
}

func (t *table) to_map() map[string]*Stats {
  m := make(map[string]*Stats, len(t.used))
  for _, i := range t.used {
    e := &t.entries[i]
    m[string(e.name)] = &e.Stats
  }
  return m
}
