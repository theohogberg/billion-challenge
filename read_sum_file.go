package main

//import "unicode/utf8"
import "time"
import "log"
import "io"
import "os"
import "fmt"

func main() {
  start := time.Now()

	file, err := os.Open("weather_stations.csv")
	if err != nil {
		log.Fatal(err) // exit
	}

  // stat the file
  file_info, _ := file.Stat()
  print("filesize: ", file_info.Size(), "\n")

  //var city_temp map[string]float32
  //var size int = 0
  var b [32]byte // ~128kb
  var re []byte // rewind buffer
  var chunk int = 0
  for {
    _, err := file.Read(b[:])
    if err == io.EOF {
      break;
    }
    if err != nil {
      log.Fatal(err) // exit
    }

    //var temp float32
    print("\n ..starting loop.. \n")
    var x int = 0
    for i := 0; i < len(b); i++ {
      c := b[i]
      // a bit too complex just simplify..
      if (c == ';') {
        // allocate the string extracted
        var name string = ""
        for ;x < i; x++ {
          c = b[x]
          // check for utf8 strings
          if (c >> 5) == 0b110 {
            d := b[x+1]
            name += string([]byte{c, d})
            //fmt.Printf("char: %s\n", name)
          }
          name += string([]byte{c})
        }
        fmt.Printf("name: %s\n", name)
        // check if ; was last char
      }
      if (c == '\n') {
        x = i+1
        re = b[x:]
        fmt.Printf("rewind: %s\n", re)
        // rewind the last buffer '\n' was last the char
      }
    }
    chunk++
  }

  elapsed := time.Now().Sub(start)
  fmt.Printf("\ntime:%dms", elapsed.Milliseconds())
}
