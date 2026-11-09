package main

// temperatures kept as integer tenths of a degree, summing floats
// drifts (25.45 came out as 25.4499..)
type Stats struct {
  min, max int64
  sum, count int64
}

// the whole file is mmapped, a worker owns every line that *starts*
// inside [start, end). no more buffers/pread, profile had ~57% in pread
func worker(data []byte, start, end int) map[string]*Stats {
  city_temp := new_table()
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
    // hash while looking for ';', one pass over the name instead of two
    semi := pos
    h := uint64(FNV_OFFSET)
    for data[semi] != ';' {
      h ^= uint64(data[semi])
      h *= FNV_PRIME
      semi++
    }
    nl := semi+1
    for nl < len(data) && data[nl] != '\n' {
      nl++
    }
    temp := parse_temp(data[semi+1:nl])
    name := data[pos:semi]
    st := city_temp.get(name, h)
    if temp < st.min { st.min = temp }
    if temp > st.max { st.max = temp }
    st.sum += temp
    st.count++
    pos = nl+1
  }
  return city_temp.to_map()
}

// always one fractional digit: -?\d{1,2}\.\d
// so skip the dot and read it as tenths directly
func parse_temp(b []byte) int64 {
  neg := false
  if b[0] == '-' {
    neg = true
    b = b[1:]
  }
  var v int64
  for _, c := range b {
    if c != '.' {
      v = v*10 + int64(c-'0')
    }
  }
  if neg {
    return -v
  }
  return v
}
