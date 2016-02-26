package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"regexp"
	"sort"
)

// seen is the list of exit nodes (by IP address) appearing in any of
// the data files. The data files are numbered (0, 1, ...) in the
// order in which they are read and the []int contains the list of
// data files (using the index number) in which the IP address
// appears. This is a poor data structure from a space perspective and
// could be replaced with a (sparse) matrix is memory becomes a
// problem.

var seen = make(map[string][]int)

// exit is a regular expression used to extract an exit node IP
// address from a file

var exit = regexp.MustCompile("^ExitAddress ([^ ]*) ")

type keySlice []string

func (k keySlice) Len() int {
	return len(k)
}
func (k keySlice) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}
func (k keySlice) Less(i, j int) bool {
	if seen[k[i]][0] == seen[k[j]][0] {
		return len(seen[k[i]]) > len(seen[k[j]])
	}

	return seen[k[i]][0] > seen[k[j]][0]
}

// sortedSeen returns the keys of the seen map as a slice of strings so
// that the map can be accessed in a sorted order. In this case
// sorting is from 'most seen' to 'least seen' (i.e. the entries with
// the most data points appear first).
func sortedSeen() []string {
	var keys keySlice

	for k, _ := range seen {
		keys = append(keys, k)
	}

	sort.Sort(keys)
	return keys
}

// parse reads a file generated by saving the output of
// https://check.torproject.org/exit-addresses looking for the
// ExitAddress lines and extracts the IP address. The index of the
// file name is passed in as i.
//
// A sample entry in the file looks like:
//
//     ExitNode FDE193D27BA55D9AC82EC766E4AABF1699EB0C40
//     Published 2016-02-25 13:45:01
//     LastStatus 2016-02-25 14:03:22
//     ExitAddress 178.209.50.151 2016-02-25 14:07:28
func parse(i int, name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	dup := make(map[string]struct{})

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		ip := exit.FindStringSubmatch(scan.Text())
		if len(ip) == 2 {
			if _, ok := dup[ip[1]]; !ok {
				seen[ip[1]] = append(seen[ip[1]], i)
				dup[ip[1]] = struct{}{}
			}
		}
	}

	return scan.Err()
}

func main() {
	out := flag.String("out", "", "Name of PNG file to write")
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 || *out == "" {
		fmt.Println("Usage: torexit -out <png> <file> [<file>...]")
		return
	}

	for i, name := range files {
		if err := parse(i, name); err != nil {
			fmt.Printf("Failed to parse file %s: %s\n", name, err)
			return
		}
	}

	if len(seen) == 0 {
		fmt.Println("No IP addresses")
		return
	}

	width := len(seen)
	height := len(files)
	size := image.Rect(0, 0, width, height)
	rect := image.NewRGBA(size)

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			rect.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	colors := [...]color.RGBA{color.RGBA{0, 0, 255, 255},
		color.RGBA{255, 0, 0, 255},
	}

	for x, k := range sortedSeen() {
		c := colors[x%len(colors)]
		for _, y := range seen[k] {
			rect.Set(x, y, c)
		}
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Printf("Failed to create %s: %s\n", *out, err)
		return
	}
	defer f.Close()

	if err := png.Encode(f, rect.SubImage(size)); err != nil {
		fmt.Printf("Failed to write %s: %s\n", *out, err)
		return
	}
}
