package main

import "time"
import "log"
import "os"
import "fmt"
import "sync"
import "runtime"
import "sort"
import "strings"
import "math"

//align buffers with cpu cache to get faster
const BUFF_SIZE = //65535  // ~65kb/16bit buffer
                  //262143 // ~262kb/18bit buffer
                  1048575// ~1048kb/20bit buffer

// temperatures kept as integer tenths of a degree, summing floats
// drifts (25.45 came out as 25.4499..)
type Stats struct {
  min, max int64
  sum, count int64
}

// longest possible line: 100 byte name + ';' + "-99.9" + '\n'
const MAX_LINE = 107

// new useful vim commands:
// cib to change parameter
// ciB to change block
func main() {
  start := time.Now()

  // main file, override with first arg
  path := "1brc/measurements.txt"
  if len(os.Args) > 1 {
    path = os.Args[1]
  }
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err) // exit
	}

  // stat the file
  file_info, err := file.Stat()
  if err != nil {
    log.Fatal(err) // exit
  }

  workers := file_info.Size()/BUFF_SIZE
  fmt.Fprintf(os.Stderr, "using: %d workers\n", workers)

  results := make(map[string]*Stats)
  var mu sync.Mutex
  var wg sync.WaitGroup
  // one goroutine per chunk spawns >13k blocking reads -> out of OS threads
  sem := make(chan struct{}, runtime.NumCPU())
  var i int64 = 0
  for ; i <= workers ; i++ {
    city_temp := make(map[string]*Stats)
    // start parsing chunks
    wg.Add(1)
    sem <- struct{}{}
    go func(offset int64) {
      defer wg.Done()
      defer func() { <-sem }()
      parse_chunk(file, city_temp, offset)
      mu.Lock()
      merge(results, city_temp)
      mu.Unlock()
    }(BUFF_SIZE*i)
  }
  // main was exiting before any worker finished..
  wg.Wait()

  print_results(results)

  elapsed := time.Now().Sub(start)
  fmt.Fprintf(os.Stderr, "time:%dms\n", elapsed.Milliseconds())
}

func merge(into, from map[string]*Stats) {
  for city, st := range from {
    dst, ok := into[city]
    if !ok {
      into[city] = st
      continue
    }
    if st.min < dst.min { dst.min = st.min }
    if st.max > dst.max { dst.max = st.max }
    dst.sum += st.sum
    dst.count += st.count
  }
}

// java's Math.round rounds half up (towards +inf), %.1f does not.
// v is in tenths
func round(v float64) float64 {
  return math.Floor(v+0.5) / 10
}

// {Abha=-23.0/18.0/59.2, Abidjan=-16.2/26.0/67.3, ...}
func print_results(results map[string]*Stats) {
  cities := make([]string, 0, len(results))
  for city := range results {
    cities = append(cities, city)
  }
  sort.Strings(cities)

  var sb strings.Builder
  sb.WriteByte('{')
  for i, city := range cities {
    st := results[city]
    if i > 0 {
      sb.WriteString(", ")
    }
    fmt.Fprintf(&sb, "%s=%.1f/%.1f/%.1f", city, round(float64(st.min)), round(float64(st.sum)/float64(st.count)), round(float64(st.max)))
  }
  sb.WriteString("}\n")
  fmt.Print(sb.String())
}


// each chunk owns every line that *starts* inside [offset, offset+BUFF_SIZE).
// read a bit past the end so the last line can be finished, and skip the
// partial line at the start (the previous chunk owns it)
func parse_chunk(file *os.File, city_temp map[string]*Stats, offset int64) {
  var buff [BUFF_SIZE+MAX_LINE]byte
  n, _ := file.ReadAt(buff[:], offset)
  data := buff[:n]

  pos := 0
  if offset != 0 {
    // the line starting exactly at offset belongs to us only if
    // the byte before it is a newline
    var prev [1]byte
    file.ReadAt(prev[:], offset-1)
    if prev[0] != '\n' {
      for pos < len(data) && data[pos] != '\n' {
        pos++
      }
      pos++
    }
  }

  for pos < len(data) && pos < BUFF_SIZE {
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
