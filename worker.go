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
    temp, next := parse_temp(data, semi+1)
    name := data[pos:semi]
    st := city_temp.get(name, h)
    if temp < st.min { st.min = temp }
    if temp > st.max { st.max = temp }
    st.sum += temp
    st.count++
    pos = next
  }
  return city_temp.to_map()
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
