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

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		ip := exit.FindStringSubmatch(scan.Text())
		if len(ip) == 2 {
			seen[ip[1]] = append(seen[ip[1]], i)
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

	width := len(seen) * 2
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

	col := 0
	for _, ys := range seen {
		c := colors[(col/2)%len(colors)]
		for _, y := range ys {
			rect.Set(col, y, c)
			rect.Set(col+1, y, c)
		}
		col += 2
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
