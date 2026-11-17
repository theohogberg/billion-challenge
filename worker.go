package main

import "encoding/binary"
import "math/bits"

// temperatures kept as integer tenths of a degree, summing floats
// drifts (25.45 came out as 25.4499..)
type Stats struct {
  min, max int64
  sum, count int64
}

const ONES  = 0x0101010101010101
const HIGHS = 0x8080808080808080
const SEMIS = 0x3B3B3B3B3B3B3B3B // ';' in every byte

// little endian 8 byte load, zero padded past the end of data so names
// near the end hash the same as everywhere else
func load8(data []byte, i int) uint64 {
  if i+8 <= len(data) {
    return binary.LittleEndian.Uint64(data[i:])
  }
  var buf [8]byte
  copy(buf[:], data[i:])
  return binary.LittleEndian.Uint64(buf[:])
}

// the whole file is mmapped, a worker owns every line that *starts*
// inside [start, end). no more buffers/pread, profile had ~57% in pread
func worker(data []byte, start, end int) map[string]*Stats {
  t := new_table()
  process(data, t, start, end)
  return t.to_map()
}

func process(data []byte, city_temp *table, start, end int) {
  pos := start
  if start != 0 {
    // start one byte early: skipping up to and including the first newline
    // then also handles a line that starts exactly at start
    pos--
    for pos < len(data) && data[pos] != '\n' {
      pos++
    }
    pos++
    // no newline left in our range: previous worker owns all of it
  }

  for pos < end {
    // SWAR: look for ';' 8 bytes at a time and hash whole words
    semi := pos
    h := uint64(FNV_OFFSET)
    for {
      w := load8(data, semi)
      x := w ^ SEMIS
      found := (x - ONES) &^ x & HIGHS
      if found != 0 {
        // lowest flagged byte is the first ';' (false positives only
        // show up above a real match)
        k := bits.TrailingZeros64(found) >> 3
        if k > 0 {
          h = (h ^ (w & (1<<(k*8) - 1))) * FNV_PRIME
        }
        semi += k
        break
      }
      h = (h ^ w) * FNV_PRIME
      semi += 8
    }
    temp, next := parse_temp(data, semi+1)
    name := data[pos:semi]
    st := city_temp.get(name, h)
    if temp < st.min { st.min = temp }
    if temp > st.max { st.max = temp }
    st.sum += temp
    st.count++
    pos = next
  }
}

// always one fractional digit: -?\d{1,2}\.\d
// so there are only 4 shapes, read them directly as tenths. returns the
// temperature and the start of the next line, no separate '\n' scan needed
func parse_temp(data []byte, i int) (int64, int) {
  neg := data[i] == '-'
  if neg {
    i++
  }
  var v int64
  if data[i+1] == '.' {
    // d.d
    v = int64(data[i]-'0')*10 + int64(data[i+2]-'0')
    i += 3
  } else {
    // dd.d
    v = int64(data[i]-'0')*100 + int64(data[i+1]-'0')*10 + int64(data[i+3]-'0')
    i += 4
  }
  if neg {
    v = -v
  }
  return v, i + 1
}
