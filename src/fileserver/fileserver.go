package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"pat"
	"strconv"
	"strings"
)

const (
	API_PREFIX      = "/api/"
	POST_PREFIX     = "post/"
	DOWNLOAD_PREFIX = "download/"
	UPLOAD_PREFIX   = "upload/"
	LOGIN_PREFIX    = "login/"
	PORT            = 8080
)

var (
	ROOT_DIR     string = "root"
	CURRENT_USER user
	USERS        []user
)

type user struct {
	Username string
	Password string
}

type configFile struct {
	Config configuration `json:"config"`
	Users  []user        `json:"users"`
}

type configuration struct {
	RootDir string `json:"rootDir"`
}

/* HTTP handlers */
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	fileName := strings.Replace(r.URL.String(), API_PREFIX+DOWNLOAD_PREFIX, "", 1)
	serveFileForDownload(w, r, fileName)
	return
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		defer file.Close()
		err := uploadFileToServer(file, header)

		if err != nil {
			fmt.Fprintln(w, err.Error())
		} else {
			fmt.Fprintf(w, "File uploaded successfully")
		}
	}
	return
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	var u user

	err := json.Unmarshal(body, &u)
	if err != nil {
		fmt.Println(err.Error())
	}

	if !loginUser(u) {
		fmt.Fprintf(w, "Invalid username or password")
	}
}

func delHandler(w http.ResponseWriter, r *http.Request) {
	// TODO
	return
}

/* move to another source file? */
func uploadFileToServer(file multipart.File, header *multipart.FileHeader) (err error) {
	os.Mkdir(ROOT_DIR, 0777)
	out, err := os.Create(ROOT_DIR + string(os.PathSeparator) + header.Filename)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func serveFileForDownload(w http.ResponseWriter, r *http.Request, fileName string) {
	if hasAccess() {
		http.ServeFile(w, r, ROOT_DIR+string(os.PathSeparator)+fileName)
	}
}

func hasAccess() bool {
	// TODO
	return true
}

func loginUser(u user) bool {
	for i := 0; i < len(USERS); i++ {
		if (USERS[i].Username == u.Username) && (USERS[i].Password == u.Password) {
			CURRENT_USER = u
			return true
		}
	}
	return false
}

func loadConfiguration() {
	path, _ := os.Getwd()
	file, err := os.Open(path + string(os.PathSeparator) + "config.json")
	if err != nil {
		fmt.Println(err.Error())
		panic(fmt.Sprintf("%s", "Failed to load configuration!"))
	}

	var buffer bytes.Buffer
	io.Copy(&buffer, file)

	var configFile configFile
	// read config file
	err = json.Unmarshal(buffer.Bytes(), &configFile)
	if err != nil {
		fmt.Println(err.Error())
		panic(fmt.Sprintf("%s", "Configuration file is corrupt!"))
	} else {
		configuration := configFile.Config
		USERS = configFile.Users
		applyConfiguration(configuration)
	}
}

func applyConfiguration(config configuration) {
	ROOT_DIR = config.RootDir
}

func initServer() *pat.PatternServeMux {
	server := pat.New()
	server.Get(API_PREFIX+DOWNLOAD_PREFIX, http.HandlerFunc(downloadHandler))

	secondaryServer := pat.New()
	secondaryServer.Post(API_PREFIX+POST_PREFIX+UPLOAD_PREFIX, http.HandlerFunc(uploadHandler))
	secondaryServer.Post(API_PREFIX+POST_PREFIX+LOGIN_PREFIX, http.HandlerFunc(loginHandler))

	server.Post(API_PREFIX+POST_PREFIX, secondaryServer)
	// TODO, DELETE_PREFIX maybe?
	server.Del(API_PREFIX+DOWNLOAD_PREFIX, http.HandlerFunc(delHandler))

	return server
}

func main() {
	loadConfiguration()
	server := initServer()
	// host client
	http.Handle("/", http.FileServer(http.Dir(".."+string(os.PathSeparator)+"client")))
	// handle file requests
	http.Handle(API_PREFIX, server)
	//start listening
	http.ListenAndServe(":"+strconv.Itoa(PORT), nil)
}
