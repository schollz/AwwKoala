package main

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/jcelliott/lumber"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/schollz/cryptopasta"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/shurcooL/github_flavored_markdown"
	"golang.org/x/crypto/bcrypt"
)

var animals []string
var adjectives []string
var aboutPageText string

var log *lumber.ConsoleLogger

func init() {
	rand.Seed(time.Now().Unix())
	animalsText, _ := Asset("static/text/animals")
	animals = strings.Split(string(animalsText), ",")
	adjectivesText, _ := Asset("static/text/adjectives")
	adjectives = strings.Split(string(adjectivesText), "\n")
	log = lumber.NewConsoleLogger(lumber.TRACE)
}

func turnOffDebugger() {
	log = lumber.NewConsoleLogger(lumber.WARN)
}

func randomAnimal() string {
	return strings.Replace(strings.Title(animals[rand.Intn(len(animals)-1)]), " ", "", -1)
}

func randomAdjective() string {
	return strings.Replace(strings.Title(adjectives[rand.Intn(len(adjectives)-1)]), " ", "", -1)
}

func randomAlliterateCombo() (combo string) {
	combo = ""
	// // first determine which names are taken from program data
	// takenNames := []string{}
	// err := db.View(func(tx *bolt.Tx) error {
	// 	// Assume bucket exists and has keys
	// 	b := tx.Bucket([]byte("programdata"))
	// 	c := b.Cursor()
	// 	for k, v := c.First(); k != nil; k, v = c.Next() {
	// 		takenNames = append(takenNames, strings.ToLower(string(v)))
	// 	}
	// 	return nil
	// })
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(takenNames)
	// generate random alliteration thats not been used
	for {
		animal := randomAnimal()
		adjective := randomAdjective()
		if animal[0] == adjective[0] { //&& stringInSlice(strings.ToLower(adjective+animal), takenNames) == false {
			combo = adjective + animal
			break
		}
	}
	return
}

// is there a string in a slice?
func stringInSlice(s string, strings []string) bool {
	for _, k := range strings {
		if s == k {
			return true
		}
	}
	return false
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func contentType(filename string) string {
	switch {
	case strings.Contains(filename, ".css"):
		return "text/css"
	case strings.Contains(filename, ".jpg"):
		return "image/jpeg"
	case strings.Contains(filename, ".png"):
		return "image/png"
	case strings.Contains(filename, ".js"):
		return "application/javascript"
	}
	return "text/html"
}

func diffRebuildtexts(diffs []diffmatchpatch.Diff) []string {
	text := []string{"", ""}
	for _, myDiff := range diffs {
		if myDiff.Type != diffmatchpatch.DiffInsert {
			text[0] += myDiff.Text
		}
		if myDiff.Type != diffmatchpatch.DiffDelete {
			text[1] += myDiff.Text
		}
	}
	return text
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Debug("%s took %s", name, elapsed)
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandStringBytesMaskImprSrc prints a random string
func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

// GetLocalIP returns the local ip address
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	bestIP := ""
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return bestIP
}

// HashPassword generates a bcrypt hash of the password using work factor 14.
// https://github.com/gtank/cryptopasta/blob/master/hash.go
func HashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 14)
	return hex.EncodeToString(hash)
}

// CheckPassword securely compares a bcrypt hashed password with its possible
// plaintext equivalent.  Returns nil on success, or an error on failure.
// https://github.com/gtank/cryptopasta/blob/master/hash.go
func CheckPasswordHash(password, hashedString string) error {
	hash, err := hex.DecodeString(hashedString)
	if err != nil {
		return err
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}

func EncryptString(toEncrypt string, password string) (string, error) {
	key := sha256.Sum256([]byte(password))
	encrypted, err := cryptopasta.Encrypt([]byte(toEncrypt), &key)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(encrypted), nil
}

func DecryptString(toDecrypt string, password string) (string, error) {
	key := sha256.Sum256([]byte(password))
	contentData, err := hex.DecodeString(toDecrypt)
	if err != nil {
		return "", err
	}
	bDecrypted, err := cryptopasta.Decrypt(contentData, &key)
	return string(bDecrypted), err
}

// exists returns whether the given file or directory exists or not
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func MarkdownToHtml(s string) string {
	unsafe := blackfriday.MarkdownCommon([]byte(s))
	pClean := bluemonday.UGCPolicy()
	pClean.AllowElements("img")
	pClean.AllowAttrs("alt").OnElements("img")
	pClean.AllowAttrs("src").OnElements("img")
	pClean.AllowAttrs("class").OnElements("a")
	pClean.AllowAttrs("href").OnElements("a")
	pClean.AllowAttrs("id").OnElements("a")
	pClean.AllowDataURIImages()
	html := pClean.SanitizeBytes(unsafe)
	return string(html)
}

func GithubMarkdownToHTML(s string) string {
	return string(github_flavored_markdown.Markdown([]byte(s)))
}
func encodeToBase32(s string) string {
	return base32.StdEncoding.EncodeToString([]byte(s))
}

func decodeFromBase32(s string) (s2 string, err error) {
	bString, err := base32.StdEncoding.DecodeString(s)
	s2 = string(bString)
	return
}
