package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	DELETE_PREFIX   = "delete/"
	DOWNLOAD_PREFIX = "download/"
	UPLOAD_PREFIX   = "upload/"
	LOGIN_PREFIX    = "login/"
	SIGNUP_PREFIX   = "signup/"
	PORT            = 8080
)

var (
	ROOT_DIR     string = "root"
	CURRENT_USER user
	USERS        []user
)

/* --- types --- */
type userNotFoundError struct {
	userName string
}

func (err *userNotFoundError) Error() string {
	return err.userName + " does not exist"
}

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

/* --- HTTP handlers --- */

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

func signupHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: extract function
	body, _ := ioutil.ReadAll(r.Body)
	var newUser user

	err := json.Unmarshal(body, &newUser)
	if err != nil {
		fmt.Println(err.Error())
	}

	e := signupUser(newUser)
	if e != nil {
		fmt.Fprintf(w, "User "+newUser.Username+" exists")
	}
}

func delHandler(w http.ResponseWriter, r *http.Request) {
	fileName := strings.Replace(r.URL.String(), API_PREFIX+DELETE_PREFIX, "", 1)
	err := os.Remove(ROOT_DIR + string(os.PathSeparator) + fileName)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
	return
}

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
	user, err := getUser(u.Username)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	if user.Password == u.Password {
		CURRENT_USER = user
		return true
	}

	return false
}

func getUser(userName string) (user, error) {
	var u user
	for _, user := range USERS {
		if user.Username == userName {
			u = user
			return u, nil
		}
	}
	var err = new(userNotFoundError)
	return u, err
}

func signupUser(newUser user) error {
	_, e := getUser(newUser.Username)
	if e == nil {
		return e
	}
	configFile := loadConfiguration()
	newUsersArray := append(configFile.Users, newUser)
	configFile.Users = newUsersArray

	jsonConfiguration, err := json.Marshal(configFile)
	if err != nil {
		log.Fatal(err.Error())
	}
	ioutil.WriteFile("config.json", jsonConfiguration, 0777)
	USERS = newUsersArray
	return nil
}

func loadConfiguration() configFile {
	path, _ := os.Getwd()
	file, err := os.Open(path + string(os.PathSeparator) + "config.json")
	defer file.Close()
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal("Failed to load configuration!")
	}

	var buffer bytes.Buffer
	io.Copy(&buffer, file)

	var configFile configFile
	// read config file
	err = json.Unmarshal(buffer.Bytes(), &configFile)
	if err != nil {
		fmt.Println(err.Error())
		log.Fatal("Configuration file is corrupt!")
	}
	return configFile
}

func applyConfiguration(config configuration) {
	ROOT_DIR = config.RootDir
}

func initServer() http.Handler {
	server := pat.New()
	server.Get(API_PREFIX+DOWNLOAD_PREFIX, http.HandlerFunc(downloadHandler))

	secondaryHandler := pat.New()
	secondaryHandler.Post(API_PREFIX+POST_PREFIX+UPLOAD_PREFIX, http.HandlerFunc(uploadHandler))
	secondaryHandler.Post(API_PREFIX+POST_PREFIX+LOGIN_PREFIX, http.HandlerFunc(loginHandler))
	secondaryHandler.Post(API_PREFIX+POST_PREFIX+SIGNUP_PREFIX, http.HandlerFunc(signupHandler))

	server.Post(API_PREFIX+POST_PREFIX, secondaryHandler)
	server.Del(API_PREFIX+DELETE_PREFIX, http.HandlerFunc(delHandler))

	return server
}

func main() {
	// init configuration
	configFile := loadConfiguration()
	configuration := configFile.Config
	USERS = configFile.Users
	applyConfiguration(configuration)

	// host client
	http.Handle("/", http.FileServer(http.Dir(".."+string(os.PathSeparator)+"client")))
	// handle file requests
	http.Handle(API_PREFIX, initServer())
	//start listening
	http.ListenAndServe(":"+strconv.Itoa(PORT), nil)
}
