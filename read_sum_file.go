package main

//import "unicode/utf8"
import "time"
import "log"
import "os"
import "fmt"
//import "strconv"


const BUFF_SIZE = 665535  // ~65kb align with cpu cache to get faster
                  //524287  // ~524kb use as test for large files

// new useful vim commands:
// cib to change parameter
// ciB to change block
func main() {
  start := time.Now()

	file, err := os.Open("weather_stations.csv")
	if err != nil {
		log.Fatal(err) // exit
	}

  // stat the file
  file_info, _ := file.Stat()
  print("filesize: ", file_info.Size(), "\n")

  var city_temp map[string]string = make(map[string]string)
  var buff  [BUFF_SIZE]byte
  var rest  []byte
  //var ct_chan = make(chan map[string]string, 200)
  run := true
  for chunk := 0; run; chunk++ {
    buff = [BUFF_SIZE]byte{} // <- I wasnt clearing the buffer
    // starting next chunk
    //fmt.Printf("\n@@@\nstarting chunk: %d\n@@@\n\n", chunk)
    // read the buffer
    n, _ := file.Read(buff[:])
    if n < BUFF_SIZE {
      fmt.Printf("END BUFF:%s\n", buff)
      run = false
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
  //fmt.Printf("%v\n",city_temp)

  elapsed := time.Now().Sub(start)
  fmt.Printf("\ntime:%dms", elapsed.Milliseconds())
}


//func parse_input
