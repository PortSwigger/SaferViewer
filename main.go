package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/pat"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

var inputPath, outputFile string

const folderName = "SaferViewer"       // $GDRIVE/$DIR where to store files
const logfile = "/tmp/SaferViewer.log" // debugging info

//go:embed client_secret.json
var secretConfig []byte

//go:embed responsefile.html
var htmlResp []byte

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		log.Fatalf("FATAL: Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		getTokenFromWeb(config)
		// we will os.Exit from above function
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) {
	// We now need to get the user to login, and authenticate our app so we can
	// continue to gdrive...

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	cmd := exec.Command("open", authURL)
	if err := cmd.Run(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
	// Spin up a web server on localhost to answer the callback from google
	p := pat.New()

	s := &http.Server{
		Addr:           "localhost:31338",
		Handler:        p,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	p.Get("/", func(res http.ResponseWriter, req *http.Request) {
		code := strings.TrimPrefix(req.RequestURI, "/?state=state-token&code=")
		code = strings.TrimSuffix(code, "&scope=https://www.googleapis.com/auth/drive")

		tok, err := config.Exchange(oauth2.NoContext, code)
		if err != nil {
			log.Fatalf("FATAL: Unable to retrieve token from web %v", err)
		}
		cacheFile, err := tokenCacheFile()
		if err != nil {
			log.Fatalf("FATAL: Unable to get path to cached credential file. %v", err)
		}
		saveToken(cacheFile, tok)
		res.WriteHeader(http.StatusOK)
		// This is embedded in from response.html at build time.
		// It should be made prettier.
		res.Write(htmlResp)
		go func() {
			if err := s.Shutdown(context.Background()); err != nil {
				log.Printf("ERROR: %v", err)
			}
			os.Exit(0)
		}()

	})

	log.Fatal(s.ListenAndServe())

}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		// return "", err
		return usr.HomeDir, err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".SaferViewer")
	//tokenCacheDir := ".credentials"
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("drive-api-cert.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) error {
	log.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
	return nil
}

// FileSizeFormat returns prettier units
func FileSizeFormat(bytes int64, forceBytes bool) string {
	if forceBytes {
		return fmt.Sprintf("%v B", bytes)
	}

	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}

	var i int
	value := float64(bytes)

	for value > 1000 {
		value /= 1000
		i++
	}
	return fmt.Sprintf("%.1f %s", value, units[i])
}

// MeasureTransferRate returns a formatted tx rate
func MeasureTransferRate() func(int64) string {
	start := time.Now()

	return func(bytes int64) string {
		seconds := int64(time.Now().Sub(start).Seconds())
		if seconds < 1 {
			return fmt.Sprintf("%s/s", FileSizeFormat(bytes, false))
		}
		bps := bytes / seconds
		return fmt.Sprintf("%s/s", FileSizeFormat(bps, false))
	}
}

func getOrCreateFolder(d *drive.Service, folderName string) string {
	folderID := ""
	if folderName == "" {
		return ""
	}
	q := fmt.Sprintf("title=\"%s\" and mimeType=\"application/vnd.google-apps.folder\"", folderName)

	r, err := d.Files.List().Q(q).MaxResults(1).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve foldername %v", err)
	}

	if len(r.Items) > 0 {
		folderID = r.Items[0].Id
	} else {
		// no folder found create new
		log.Printf("Folder not found. Create new folder : %s\n", folderName)
		f := &drive.File{Title: folderName, Description: "SaferViewer Cache directory", MimeType: "application/vnd.google-apps.folder"}
		r, err := d.Files.Insert(f).Do()
		if err != nil {
			log.Printf("An error occurred when create folder: %v\n", err)
		}
		folderID = r.Id
	}
	return folderID
}

func uploadFile(d *drive.Service, title string, description string,
	parentName string, mimeType string, filename string) (*drive.File, error) {
	input, err := os.Open(filename)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, err
	}
	// Grab file info
	inputInfo, err := input.Stat()
	if err != nil {
		return nil, err
	}

	parentID := getOrCreateFolder(d, parentName)

	f := &drive.File{Title: title, Description: description, MimeType: mimeType}
	if parentID != "" {
		p := &drive.ParentReference{Id: parentID}
		f.Parents = []*drive.ParentReference{p}
	}
	getRate := MeasureTransferRate()

	// progress call back
	showProgress := func(current, total int64) {
		log.Printf("INFO Uploaded at %s", getRate(current))
	}

	r, err := d.Files.Insert(f).ResumableMedia(context.Background(), input, inputInfo.Size(), mimeType).ProgressUpdater(showProgress).Do()
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, err
	}

	return r, nil
}

func main() {
	/* Log better */
	log.SetFlags(log.LstdFlags | log.Ldate | log.Lmicroseconds | log.Lshortfile)
	f, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	//defer to close when you're done with it, not because you think it's idiomatic!
	defer f.Close()

	//set output of logs to f
	log.SetOutput(f)
	log.Printf("INFO: Starting")

	if len(os.Args) == 1 {
		log.Printf("ERROR: No command line arguments supplied")
		log.Printf("INFO: This application only operates in Drag and Drop mode!")
		os.Exit(1)
	}

	ctx := context.Background()

	config, err := google.ConfigFromJSON(secretConfig, drive.DriveScope)
	if err != nil {
		log.Fatalf("FATAL: Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("FATAL: Unable to retrieve drive Client %v", err)
	}

	inputPath = os.Args[1]
	log.Printf("INFO: Read file: %s\n", inputPath)
	outputTitle := outputFile
	if outputTitle == "" {
		outputTitle = filepath.Base(inputPath)
	}
	log.Printf("INFO: Output name: %s\n", outputTitle)

	ext := filepath.Ext(inputPath)
	mimeType := "application/octet-stream"
	if ext != "" {
		mimeType = mime.TypeByExtension(ext)
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	log.Printf("INFO: Mime is %s\n", mimeType)

	file, err := uploadFile(srv, outputTitle, "", folderName, mimeType, inputPath)
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	log.Printf("INFO: Document Link is %#v", file.EmbedLink)

	cmd := exec.Command("open", file.EmbedLink)
	if err := cmd.Run(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}
