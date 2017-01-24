// Complete static page web server written in Go.
// https://github.com/golang/go/wiki/HttpStaticFiles

package main

import "net/http"
import "flag"
import "fmt"
import "os"
import "time"

var (
	port    int
	seconds int
)

// var path string

func init() {
	flag.IntVar(&port, "port", 8080, "the port; defaults to 8080")
	//flag.StringVar(&path, "webroot", ".", "the webroot, where the html files are; defaults to the current/working directory")
	flag.IntVar(&seconds, "seconds", 0, "the number of seconds to leave server running. set to 0 to run indefinitely; defaults to 0")
	flag.Parse()
}

func main() {
	// if path == "." {
	path, err := os.Getwd()
	if err != nil {
		fmt.Println("failed to get current directory.")
		return
	}
	//     path = curpath
	// }

	// TODO: Read addtl config options from the .gostatic.conf file if it exists..

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("cannot find (or can't access) html root path.")
	}

	if seconds > 0 {
		go func() {
			time.Sleep(time.Second * time.Duration(seconds))
			os.Exit(0)
		}()
	}

	fmt.Printf("gostatic is listening: http://localhost:%d/\n", port)

	panic(http.ListenAndServe(fmt.Sprintf(":%d", port), http.FileServer(http.Dir(path))))
}
