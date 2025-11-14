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

  var semi_i int = 0
  var semi [999]int
  //var size int = 0
  var b [32]byte // ~128kb
  var chunk int = 0
  for {
    _, err := file.Read(b[:])
    if err == io.EOF {
      break;
    }
    if err != nil {
      log.Fatal(err) // exit
    }
    //var r rune
    print("staring loop.. \n")
    for i := 0; i < len(b); i++ {
      c := b[i]
      fmt.Printf("char: %c\n", c)
      //if (c == '\n') {
      //}
      if (c == ';') {
        semi[semi_i] = i+chunk*32
        semi_i++
        fmt.Printf("i: %d\n", i+chunk*32)
      }
      // if first byte is utf expansion
      if (c >> 5) == 0b110 {
        r := b[i+1]
        fmt.Printf("char: %s\n", []byte{c, r})
      }
    }
    chunk++
  }

  elapsed := time.Now().Sub(start)
  print("\n", elapsed.Milliseconds())
}
