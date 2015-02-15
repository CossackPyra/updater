package updater

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"syscall"
	"time"
)

type updaterHandler struct {
	tempDir      string
	rand1        []byte
	key1         []byte
	execFilename string
}

func UpdaterServer(tempDir string, key1 []byte, execFilename string) http.Handler {
	rand1 := make([]byte, 16)
	rand.Read(rand1)
	return &updaterHandler{tempDir, rand1, key1, execFilename}
}

func reportError(w http.ResponseWriter, s1 string) {
	fmt.Println("reportError = " + s1)
	m1 := map[string]interface{}{"error": true, "message": s1}
	b1, _ := json.MarshalIndent(m1, "", "\t")
	w.Write(b1)
}

func (u *updaterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("updater.ServeHTTP")
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if r.Method == "GET" {
		w.Write(u.rand1)
		fmt.Println(hex.EncodeToString(u.rand1))
		return
	}
	f1, err := ioutil.TempFile(u.tempDir, "upX17")
	if err != nil {
		reportError(w, "Can't create temp file upX17 1")
		return
	}
	defer func() {
		f1.Close()
	}()
	// f2, err := ioutil.TempFile(u.tempDir, "upX18")
	// if err != nil {
	// 	reportError(w, "Can't create temp file upX18 1")
	// 	return
	// }
	// defer func() {
	// 	f2.Close()
	// }()
	// if r.Method != "POST" {
	// 	reportError(w, "Bad Method")
	// 	return
	// }

	// n, err := io.Copy(f1, r.Body)
	// if err != nil {
	// 	reportError(w, "Failed to copy: "+err.Error())
	// 	return
	// }

	block, err := aes.NewCipher([]byte(u.key1))
	if err != nil {
		reportError(w, "Failed init cipher: "+err.Error())
		return
	}

	stream := cipher.NewCFBDecrypter(block, u.rand1)

	hash1 := make([]byte, 20)
	_, err = r.Body.Read(hash1)
	if err != nil {
		reportError(w, "Failed read hash1: "+err.Error())
		return
	}
	stream.XORKeyStream(hash1[:], hash1[:])

	h := sha1.New()

	buf := make([]byte, 32*1024)
	var written int64
	for {
		nr, er := r.Body.Read(buf)
		if nr > 0 {

			stream.XORKeyStream(buf[:nr], buf[:nr])
			h.Write(buf[0:nr])
			nw, ew := f1.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				reportError(w, "Failed write: "+ew.Error())
				return
			}
			if nr != nw {
				reportError(w, "Short write")
				return
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			reportError(w, "Failed read: "+er.Error())
			return
		}
	}

	hash2 := h.Sum(nil)

	verified := bytes.Equal(hash1, hash2)

	if !verified {
		reportError(w, "Hash failed")
		return
	}

	f1.Close()

	err = os.Rename(f1.Name(), u.execFilename)

	if err != nil {
		reportError(w, "Failed move: "+err.Error())
		return
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		// os.Remove(os.Args[0])
		// ioutil.WriteFile(os.Args[0], b, 0700)
		os.Chmod(u.execFilename, 0700)
		fmt.Println("updater.ServeHTTP RUN")
		syscall.Exec(u.execFilename, []string{u.execFilename}, []string{})
	}()
	fmt.Println("updater.ServeHTTP END")

}

func PostFile(url string, filename string, key1 []byte) error {

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("err 100")
		return err
	}
	rand1, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("err 200")
		return err
	}

	fmt.Println("Got rand1: " + hex.EncodeToString(rand1))

	block, err := aes.NewCipher([]byte(key1))
	if err != nil {
		fmt.Println("err 300")
		return err
	}

	stream := cipher.NewCFBEncrypter(block, rand1)

	b1, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("err 400")
		return err
	}

	hash1 := sha1.Sum(b1)

	buf := new(bytes.Buffer)
	buf.Write(hash1[:])
	buf.Write(b1)
	b1 = buf.Bytes()

	stream.XORKeyStream(b1, b1)

	buf = bytes.NewBuffer(b1)

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		fmt.Println("err 500")
		return err
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	b2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("err 600")
		return err
	}
	var m1 map[string]interface{}
	err = json.Unmarshal(b2, &m1)
	if err != nil {
		fmt.Println("err 700")
		return err
	}
	if m1["error"] == true {
		fmt.Println("err 800")
		return errors.New(fmt.Sprintf("Remote Error: %v", m1["message"]))
	}
	return nil
}
