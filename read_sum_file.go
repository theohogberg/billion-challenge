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
import "flag"
import "runtime/pprof"

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

  cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
  flag.Parse()

  if *cpuprofile != "" {
    f, err := os.Create(*cpuprofile)
    if err != nil {
      log.Fatal(err)
    }
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
  }

  // main file, override with first arg
  path := "1brc/measurements.txt"
  if flag.NArg() > 0 {
    path = flag.Arg(0)
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

  // one long lived worker per cpu, each owning a contiguous range of
  // the file and reusing its buffer + map (was 13k tiny tasks + merges)
  size := file_info.Size()
  workers := int64(runtime.NumCPU())
  per_worker := size/workers + 1
  fmt.Fprintf(os.Stderr, "using: %d workers\n", workers)

  partials := make([]map[string]*Stats, workers)
  var wg sync.WaitGroup
  for i := int64(0); i < workers; i++ {
    wg.Add(1)
    go func(i int64) {
      defer wg.Done()
      start := i*per_worker
      end := min(start+per_worker, size)
      partials[i] = worker(file, start, end)
    }(i)
  }
  wg.Wait()

  results := make(map[string]*Stats)
  for _, p := range partials {
    merge(results, p)
  }

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
