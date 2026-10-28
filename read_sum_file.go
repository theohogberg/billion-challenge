package main

import "time"
import "log"
import "os"
import "fmt"
import "sync"
import "runtime"
import "strconv"

//align buffers with cpu cache to get faster
const BUFF_SIZE = //65535  // ~65kb/16bit buffer
                  //262143 // ~262kb/18bit buffer
                  1048575// ~1048kb/20bit buffer

type Stats struct {
  min, max, sum float64
  count int64
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
  fmt.Printf("using: %d workers\n", workers)

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
    }(BUFF_SIZE*i)
  }
  // main was exiting before any worker finished..
  wg.Wait()

  elapsed := time.Now().Sub(start)
  fmt.Printf("\ntime:%dms\n", elapsed.Milliseconds())
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
    city := string(data[pos:semi])
    temp, err := strconv.ParseFloat(string(data[semi+1:nl]), 64)
    if err != nil {
      log.Fatal(err)
    }
    st, ok := city_temp[city]
    if !ok {
      st = &Stats{min: temp, max: temp}
      city_temp[city] = st
    }
    if temp < st.min { st.min = temp }
    if temp > st.max { st.max = temp }
    st.sum += temp
    st.count++
    pos = nl+1
  }
}
