package main

import "time"
import "log"
import "os"
import "fmt"
import "sync"
import "runtime"
import "flag"
import "runtime/pprof"

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
