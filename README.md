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
- per-worker open addressing table (FNV-1a, linear probing) keyed on slices
  of the mmapped file, so station names are never copied in the hot loop.
  Converted to a map only for the final merge
- the name is hashed in the same loop that looks for `;`
- temperatures only come in 4 shapes (`d.d`, `dd.d`, `-d.d`, `-dd.d`) so they
  are read directly, which also gives the start of the next line

## Timings

Full 1B rows, M1 Pro (8 cores), warm page cache.

| change                                     | time   |
|--------------------------------------------|--------|
| first working version (ParseFloat, 13k tasks) | 13.6s |
| parse temperature as int tenths            | 10.7s  |
| no string alloc on map lookup              | 6.3s   |
| one worker per CPU, reuse buffers          | 5.1s   |
| mmap instead of ReadAt                     | 5.0s   |
| custom hash table                          | 4.3s   |
| hash while scanning for `;`                | 3.8s   |
| fixed shape temperature parsing            | 3.1s   |

mmap barely moved it: the 57% that was in `pread` turned into page faults
inside the worker loop. The parsing loop itself was the next thing to look at.

`make bench` runs the worker over `weather_stations_small.txt` (100k lines),
which is a lot quicker to iterate on: 502 MB/s with the hash table, 685 MB/s
after the two parsing changes.

`testdata/` has a few extra edge cases (empty file, no trailing newline, a
single line) that `make test` runs together with the reference samples.
