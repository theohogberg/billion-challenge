package main

import "time"
import "log"
import "os"
import "fmt"
import "sync"
import "runtime"
import "flag"
import "runtime/pprof"
import "syscall"
import "sync/atomic"

const CHUNK_SIZE = 16 << 20

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

  size := int(file_info.Size())
  if size == 0 {
    // can't mmap an empty file
    print_results(map[string]*Stats{})
    return
  }
  data, err := syscall.Mmap(int(file.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
  if err != nil {
    log.Fatal(err)
  }
  defer syscall.Munmap(data)

  // workers pull fixed size chunks off a counter instead of getting
  // size/NumCPU each, so slower (efficiency) cores just take fewer chunks.
  // only ~1% on the M1 Pro (2.95s -> 2.91s), more workers than cpus is
  // slower (12: 2.97s, 16: 3.05s), 6 is 3.29s
  workers := runtime.NumCPU()
  chunks := (size + CHUNK_SIZE - 1) / CHUNK_SIZE
  fmt.Fprintf(os.Stderr, "using: %d workers, %d chunks\n", workers, chunks)

  var next atomic.Int64
  partials := make([]map[string]*Stats, workers)
  var wg sync.WaitGroup
  for i := 0; i < workers; i++ {
    wg.Add(1)
    go func(i int) {
      defer wg.Done()
      t := new_table()
      for {
        c := int(next.Add(1) - 1)
        if c >= chunks {
          break
        }
        start := c*CHUNK_SIZE
        process(data, t, start, min(start+CHUNK_SIZE, size))
      }
      partials[i] = t.to_map()
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
