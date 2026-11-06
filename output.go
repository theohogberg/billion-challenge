package main

import "fmt"
import "math"
import "sort"
import "strings"

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
