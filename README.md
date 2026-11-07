# billion-challenge

My Go take on the [One Billion Row Challenge](https://github.com/gunnarmorling/1brc):
read a 1B line `station;temperature` file and print min/mean/max per
station, sorted by name.

```
make            # build to build/brc
make test       # compare against the reference samples in 1brc/
make run        # full 1B row file (1brc/measurements.txt, ~13GB)
make profile    # cpu profile of a full run
```

Generate `measurements.txt` with the scripts in the `1brc` submodule.

## Approach

- split the file into one contiguous range per CPU
- the file is mmapped, a worker owns every line that *starts* inside its range
  (`ReadAt` in 4mb chunks before that, see history)
- temperatures are parsed straight into integer tenths, no floats until output
- per-worker `map[string]*Stats`, merged at the end

## Timings

Full 1B rows, M1 Pro (8 cores), warm page cache.

| change                                     | time   |
|--------------------------------------------|--------|
| first working version (ParseFloat, 13k tasks) | 13.6s |
| parse temperature as int tenths            | 10.7s  |
| no string alloc on map lookup              | 6.3s   |
| one worker per CPU, reuse buffers          | 5.1s   |
| mmap instead of ReadAt                     | 5.0s   |

mmap barely moved it: the 57% that was in `pread` turned into page faults
inside the worker loop. The parsing loop itself is the next thing to look at.

`testdata/` has a few extra edge cases (empty file, no trailing newline, a
single line) that `make test` runs together with the reference samples.
