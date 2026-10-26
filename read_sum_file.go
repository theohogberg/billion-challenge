package main

//import "unicode/utf8"
import "time"
import "log"
import "os"
import "fmt"
import "sync"
import "runtime"
//import "strconv"


//align buffers with cpu cache to get faster
const BUFF_SIZE = //65535  // ~65kb/16bit buffer
                  //262143 // ~262kb/18bit buffer
                  1048575// ~1048kb/20bit buffer

// new useful vim commands:
// cib to change parameter
// ciB to change block
func main() {
  start := time.Now()

  // main file
	file, err := os.Open("1brc/measurements.txt")
	//file, err := os.Open("weather_stations.txt")
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
  // make a array of city_temp strings here and use each
  // for every goroutine
  for ; i <= workers ; i++ {
    var city_temp map[string]string = make(map[string]string)
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


func parse_chunk(file *os.File, city_temp map[string]string, offset int64) {
  //fmt.Printf("\n@@@\nstarting chunk: %d\n@@@\n\n", chunk)

  var buff  [BUFF_SIZE]byte
  var rest  []byte
  // fill the buffer
  n, _ := file.ReadAt(buff[:], offset)
  // only useful for last worker..
  if n < BUFF_SIZE {
    fmt.Printf("last bytes: %d\n", n)
  }

  var next_city bool = true
  var after_semi, after_nl int = 0, 0 // chunk*BUFF_SIZE+after_nl to get pos
  var city string
  var temp string
  // add the rest from the last newline (no checks just add)
  loop_buff := append(rest, buff[:]...)
  //fmt.Println("rest:",string(rest[:]))
  rest = []byte{}
  //fmt.Println("buff:",string(buff[:]))
  //fmt.Println("loop:",string(loop_buff[:]))

  for i := 0; i < len(loop_buff); i++ {
    c := loop_buff[i]
    //if i == 0 { fmt.Println(string(c)) }
    // check for next city. if so, check for semi otherwise newline
    if next_city { goto semi } else { goto newline }

    semi:
      if (c == ';') {
        after_semi = i+1
        city = string(loop_buff[after_nl:i])
        next_city = false
      }
    continue

    newline:
      if (c == '\n') {
        after_nl = i+1
        temp = string(loop_buff[after_semi:i])
        next_city = true
        rest = loop_buff[after_nl:]
        // newline hit lets save
        city_temp[city] = temp
      }
  }
}
