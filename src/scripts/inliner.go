package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	out, err := os.Create("inline.go")
	ok(err)

	fmt.Fprintln(out, "package main")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "var inline = map[string]string {")

	ok(filepath.Walk("inline", func(path string, info os.FileInfo, err error) error {
		ok(err)

		if !info.IsDir() {
			fmt.Fprintf(out, "    \"%s\": `", path[7:])

			f, err := os.Open(path)
			ok(err)

			reader := bufio.NewReader(f)

			for {
				bytes, readErr := reader.ReadSlice('`')

				_, writeErr := out.Write(bytes)
				ok(writeErr)

				if readErr == io.EOF {
					break
				}

				ok(readErr)

				_, err := out.WriteString(" + \"`\" + `")
				ok(err)
			}

			fmt.Fprintln(out, "`,")
			fmt.Fprintln(out)
			f.Close()
		}

		return nil
	}))

	fmt.Fprintln(out, "}")
	out.Close()
}

func ok(err error) {
	if err != nil {
		panic(err)
	}
}
