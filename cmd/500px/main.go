package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/orijtech/500px/v1"
	"github.com/orijtech/otils"

	"github.com/skratchdot/open-golang/open"
)

func parser(args []string) error {
	var firstArg string
	var rest []string
	if len(args) >= 2 {
		firstArg, rest = args[1], args[2:]
	}

	switch firstArg {
	default:
		return fmt.Errorf("unknown command %q", firstArg)
	case "upload":
		return upload(rest)
	case "init":
		return initOAuth(rest)
	}
}

func initOAuth(args []string) error {
	token, err := px500.OAuth1AuthorizationByEnv()
	if err != nil {
		return err
	}

	kvMapping := []struct {
		key   string
		value string
	}{
		{"PX500_ACCESS_SECRET", token.TokenSecret},
		{"PX500_ACCESS_TOKEN", token.Token},
	}

	fmt.Printf("Please set in your environment the keys below:\n")
	for _, kv := range kvMapping {
		fmt.Printf("\t%s=%s\n", kv.key, kv.value)
	}

	return nil
}

type uploadCmd struct {
	iso    string
	title  string
	path   string
	stdin  bool
	tagStr string

	private bool
	nsfw    bool

	description string
}

func useOrMakeTitle(title string) string {
	if title != "" {
		return title
	}
	return fmt.Sprintf("Upload from terminal at %v", time.Now())
}

var errEitherPathOrStdin = errors.New("either `path` or `stdin` have to be set")

func (ucmd *uploadCmd) validate() error {
	if ucmd.path == "" && !ucmd.stdin {
		return errEitherPathOrStdin
	}
	return nil
}

func upload(args []string) error {
	client, err := px500.NewOAuth1ClientFromEnv()
	if err != nil {
		log.Printf("Perhaps try running command: `init`")
		return err
	}

	ucmd := new(uploadCmd)
	if err := ucmd.parse(args); err != nil {
		return err
	}

	if err := ucmd.validate(); err != nil {
		return err
	}

	var f io.Reader = os.Stdin
	title := ucmd.title
	if ucmd.path != "" {
		ff, err := os.Open(ucmd.path)
		if err != nil {
			return err
		}
		f = ff
		defer ff.Close()

		if title == "" {
			title = filepath.Base(ucmd.path)
		}
	}

	photo, err := client.UploadPhoto(&px500.UploadRequest{
		Body: f,
		PhotoInfo: &px500.Photo{
			Title:       otils.NullableString(useOrMakeTitle(title)),
			ISO:         otils.NullableString(ucmd.iso),
			Tags:        strings.Split(ucmd.tagStr, ","),
			Private:     ucmd.private,
			Description: otils.NullableString(ucmd.description),
			NSFW:        ucmd.nsfw,
		},
	})
	if err != nil {
		return err
	}

	photoURL := makeURL(photo)
	return open.Start(photoURL)
}

func makeURL(photo *px500.Photo) string {
	return fmt.Sprintf("https://500px.com/photo/%v", photo.ID)
}

func (ucmd *uploadCmd) parse(args []string) error {
	fset := flag.NewFlagSet("upload", flag.ExitOnError)
	fset.StringVar(&ucmd.path, "path", "", "the path containing the photo")
	fset.StringVar(&ucmd.description, "description", "uploaded from 500px CLI", "the description for the photo")
	fset.BoolVar(&ucmd.stdin, "stdin", false, "whether to read the file from standard input")
	fset.StringVar(&ucmd.title, "title", "", "the title of the photo")
	fset.StringVar(&ucmd.tagStr, "tags", "", "tags separated by commas e.g photos,selfies,2017")
	fset.StringVar(&ucmd.iso, "iso", "", "the ISO of the camera used to take the photo")
	fset.BoolVar(&ucmd.nsfw, "nsfw", false, "set the photo as NSFW(Not Safe For Work)")
	fset.BoolVar(&ucmd.private, "private", false, "make the photo private by default")
	return fset.Parse(args)
}

func main() {
	if err := parser(os.Args); err != nil {
		log.Fatal(err)
	}
}
