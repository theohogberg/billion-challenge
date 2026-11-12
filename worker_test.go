package main

import "os"
import "testing"

func TestParseTemp(t *testing.T) {
  cases := map[string]int64{
    "0.0":   0,
    "1.5":   15,
    "-1.5":  -15,
    "12.3":  123,
    "-99.9": -999,
    "99.9":  999,
    "-0.1":  -1,
  }
  for in, want := range cases {
    got, next := parse_temp([]byte(in+"\n"), 0)
    if got != want {
      t.Errorf("parse_temp(%q) = %d, want %d", in, got, want)
    }
    if next != len(in)+1 {
      t.Errorf("parse_temp(%q) next = %d, want %d", in, next, len(in)+1)
    }
  }
}

func TestRound(t *testing.T) {
  cases := []struct {
    tenths float64
    want   float64
  }{
    {254.5, 25.5},  // half up
    {-254.5, -25.4}, // half up means towards +inf for negatives too
    {12.4, 1.2},
    {0, 0},
  }
  for _, c := range cases {
    if got := round(c.tenths); got != c.want {
      t.Errorf("round(%v) = %v, want %v", c.tenths, got, c.want)
    }
  }
}

// split the input at every possible byte and make sure two workers
// together see exactly what one worker sees
func TestWorkerSplit(t *testing.T) {
  data := []byte("Hamburg;12.0\nBulawayo;8.9\nPalembang;38.8\nHamburg;34.2\nCracow;-12.6\n")
  whole := worker(data, 0, len(data))

  for split := 0; split <= len(data); split++ {
    got := make(map[string]*Stats)
    merge(got, worker(data, 0, split))
    merge(got, worker(data, split, len(data)))

    if len(got) != len(whole) {
      t.Fatalf("split at %d: %d stations, want %d", split, len(got), len(whole))
    }
    for city, w := range whole {
      g, ok := got[city]
      if !ok || *g != *w {
        t.Fatalf("split at %d: %s = %+v, want %+v", split, city, g, w)
      }
    }
  }
}

// ~100k lines, quicker to iterate on than the full file
func BenchmarkWorker(b *testing.B) {
  data, err := os.ReadFile("weather_stations_small.txt")
  if err != nil {
    b.Skip(err)
  }
  b.SetBytes(int64(len(data)))
  b.ResetTimer()
  for i := 0; i < b.N; i++ {
    worker(data, 0, len(data))
  }
}
