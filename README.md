# CossackPyra/updater

To update program written in Go you should usually stop program, upload new executable and run it.
Programs in Go often implement web services and I designed `http.Handler` implementation that update it self.

Here is what it does:-

1. accepts AES encrypted executable via POST
2. decrypts using secret and AES
3. verifies hash
4. overwrites itself and executes new file.

It may work relatively secure without `HTTPS`

Here is web service that implments self updating

    package main
    
    import (
    	"fmt"
    	"io"
    	"net/http"
    	"os"
    	"github.com/CossackPyra/updater"
    )
    
    func main() {
    
    	http.HandleFunc("/", handle_def)
    
    	os.Mkdir("tmp", 0700)
    	u1 := updater.UpdaterServer("tmp", []byte("1234567890123456"), os.Args[0])
    	http.Handle("/updater-me", u1)
    	http.ListenAndServe(":9999", nil)
    }
    
    func handle_def(w http.ResponseWriter, r *http.Request) {
    	fmt.Println("def ", r.Method)
    	io.WriteString(w, "v 24\n")
    	io.WriteString(w, r.URL.Path)
    	println(r.URL.Path)
    }


`[]byte("1234567890123456")` is secret

`31323334353637383930313233343536` is hexadecimal encoding of `[]byte("1234567890123456")`

`new-service` is new executable that we wish to upload instead of old and run

    pyra-poster 31323334353637383930313233343536 new-service http://127.0.0.1:9999/updater-me


__Package Intallation__

    go get github.com/CossackPyra/updater

__pyra-poster Installation__

    go get github.com/CossackPyra/updater/pyra-poster

I designed it for Linux. I use it on servers and Raspbery Pi (gobot), current version will not work on Windows.

