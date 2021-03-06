package updater

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
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

var __debug_me bool = false

func DebugMe(b1 bool) {
	__debug_me = b1
}

func debug(s1 string) {
	if !__debug_me {
		return
	}
	fmt.Println(s1)
}

func UpdaterServer(tempDir string, key1 []byte, execFilename string) http.Handler {
	rand1 := make([]byte, 16)
	rand.Read(rand1)
	return &updaterHandler{tempDir, rand1, key1, execFilename}
}

func reportError(w http.ResponseWriter, s1 string) {
	debug("reportError = " + s1)
	m1 := map[string]interface{}{"error": true, "message": s1}
	b1, _ := json.MarshalIndent(m1, "", "\t")
	w.Write(b1)
}
func reportOk(w http.ResponseWriter) {
	m1 := map[string]interface{}{"ok": true}
	b1, _ := json.MarshalIndent(m1, "", "\t")
	w.Write(b1)
}

func (u *updaterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debug("updater.ServeHTTP")
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if r.Method == "GET" {
		w.Write(u.rand1)
		debug(hex.EncodeToString(u.rand1))
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

	rand0 := make([]byte, 20)
	_, err = r.Body.Read(rand0)
	if err != nil {
		reportError(w, "Failed read rand0: "+err.Error())
		return
	}
	stream.XORKeyStream(rand0[:], rand0[:])
	debug("rand0: " + hex.EncodeToString(rand0))

	hash1 := make([]byte, 20)
	_, err = r.Body.Read(hash1)
	if err != nil {
		reportError(w, "Failed read hash1: "+err.Error())
		return
	}
	stream.XORKeyStream(hash1[:], hash1[:])

	{
		// header
		buf1 := new(bytes.Buffer)
		io.WriteString(buf1, "pyra-poster")
		binary.Write(buf1, binary.LittleEndian, int16(1))
		binary.Write(buf1, binary.LittleEndian, int32(0))
		bx1 := buf1.Bytes()
		bx2 := make([]byte, len(bx1))
		_, err = r.Body.Read(bx2)
		if err != nil {
			reportError(w, "Failed read header: "+err.Error())
			return
		}
		stream.XORKeyStream(bx2[:], bx2[:])
		debug("header: " + string(bx2))
		if !bytes.Equal(bx2, bx1) {
			reportError(w, "Failed wrong header")
			return
		}
	}

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
		debug("updater.ServeHTTP RUN")
		syscall.Exec(u.execFilename, []string{u.execFilename}, []string{})
	}()
	debug("updater.ServeHTTP END")

	reportOk(w)
}

func PostFile(url string, filename string, key1 []byte) error {

	resp, err := http.Get(url)
	if err != nil {
		debug("err 100")
		return err
	}
	rand1, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		debug("err 200")
		return err
	}

	debug("Got rand1: " + hex.EncodeToString(rand1))

	block, err := aes.NewCipher([]byte(key1))
	if err != nil {
		debug("err 300")
		return err
	}

	stream := cipher.NewCFBEncrypter(block, rand1)

	b1, err := ioutil.ReadFile(filename)
	if err != nil {
		debug("err 400")
		return err
	}

	hash1 := sha1.Sum(b1)

	buf := new(bytes.Buffer)

	rand0 := make([]byte, 20)
	rand.Read(rand0)
	buf.Write(rand0)
	debug("rand0: " + hex.EncodeToString(rand0))

	buf.Write(hash1[:])
	io.WriteString(buf, "pyra-poster")
	binary.Write(buf, binary.LittleEndian, int16(1))
	binary.Write(buf, binary.LittleEndian, int32(0))

	buf.Write(b1)
	b1 = buf.Bytes()

	stream.XORKeyStream(b1, b1)

	buf = bytes.NewBuffer(b1)

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		debug("err 500")
		return err
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	b2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		debug("err 600")
		return err
	}
	var m1 map[string]interface{}
	err = json.Unmarshal(b2, &m1)
	if err != nil {
		debug("err 700")
		return err
	}
	if m1["error"] == true {
		debug("err 800")
		return errors.New(fmt.Sprintf("Remote Error: %v", m1["message"]))
	}
	return nil
}
