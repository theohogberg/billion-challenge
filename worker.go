package main

import "os"

//align buffers with cpu cache to get faster
// measured on the full file (M1 Pro, 8 workers):
//   128kb ~5.6s, 256kb ~5.3s, 1mb ~5.1s, 4mb ~5.05s
// so it barely matters, the pread copy is the real cost
const BUFF_SIZE = 4194303 // ~4mb/22bit buffer

// temperatures kept as integer tenths of a degree, summing floats
// drifts (25.45 came out as 25.4499..)
type Stats struct {
  min, max int64
  sum, count int64
}

// longest possible line: 100 byte name + ';' + "-99.9" + '\n'
const MAX_LINE = 107

func worker(file *os.File, start, end int64) map[string]*Stats {
  city_temp := make(map[string]*Stats)
  buff := make([]byte, BUFF_SIZE+MAX_LINE+1)
  for off := start; off < end; off += BUFF_SIZE {
    parse_chunk(file, buff, city_temp, off, min(BUFF_SIZE, end-off))
  }
  return city_temp
}

// each chunk owns every line that *starts* inside [offset, offset+limit).
// read a bit past the end so the last line can be finished, and skip the
// partial line at the start (the previous chunk owns it)
func parse_chunk(file *os.File, buff []byte, city_temp map[string]*Stats, offset, limit int64) {
  pos := 0
  end := int(limit)
  if offset != 0 {
    // start one byte early: skipping up to and including the first newline
    // then also handles a line that starts exactly at offset
    offset--
    end++
    n, _ := file.ReadAt(buff, offset)
    buff = buff[:n]
    for buff[pos] != '\n' {
      pos++
    }
    pos++
  } else {
    n, _ := file.ReadAt(buff[1:], offset)
    buff = buff[1:n+1]
  }
  data := buff

  for pos < len(data) && pos < end {
    semi := pos
    for data[semi] != ';' {
      semi++
    }
    nl := semi+1
    for nl < len(data) && data[nl] != '\n' {
      nl++
    }
    temp := parse_temp(data[semi+1:nl])
    // m[string(b)] doesn't allocate on lookup, only copy the name on insert
    st, ok := city_temp[string(data[pos:semi])]
    if !ok {
      st = &Stats{min: temp, max: temp}
      city_temp[string(data[pos:semi])] = st
    }
    if temp < st.min { st.min = temp }
    if temp > st.max { st.max = temp }
    st.sum += temp
    st.count++
    pos = nl+1
  }
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
