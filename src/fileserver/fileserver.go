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

	USER_DIRECTORY_PREFIX = "user-"
	PORT                  = 8080
)

var (
	ROOT_DIR        string = "root"
	USER_LOGIN_DATA        = make(map[string]bool)
	USERS           []user
)

/* --- types --- */
type userNotFoundError struct {
	userName string
}

func (err *userNotFoundError) Error() string {
	return err.userName + " does not exist"
}

type userExistsError struct {
	userName string
}

func (err *userExistsError) Error() string {
	return "User " + err.userName + " is already registered!"
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

type fileInfo struct {
	Name string `json:name`
	Size int64  `json:size`
}

type dirInfo struct {
	Files []fileInfo `json:files`
}

/* --- HTTP handlers --- */
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	parameters := strings.Split(strings.Replace(r.URL.String(), API_PREFIX+DOWNLOAD_PREFIX, "", 1), "&")
	userName := strings.Split(parameters[0], "=")[1]
	fileName := parameters[1]
	if hasAccess(userName) {
		serveFileForDownload(w, r, fileName)
	}
	return
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.Replace(r.URL.String(), API_PREFIX+POST_PREFIX+UPLOAD_PREFIX, "", 1)
	if !hasAccess(username) {
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		defer file.Close()
		err := uploadFileToServer(file, header, username)

		if err != nil {
			fmt.Fprintln(w, err.Error())
		} else {
			dirInfo, e := readDirectory(ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + username)
			if e != nil {
				fmt.Fprintf(w, e.Error())
			} else {
				jsonDirInfo, err := json.Marshal(dirInfo)
				if err != nil {
					log.Print(err.Error())
				} else {
					fmt.Fprintf(w, string(jsonDirInfo))
				}
			}
		}
	}
	return
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RemoteAddr)
	body, _ := ioutil.ReadAll(r.Body)
	var u user

	err := json.Unmarshal(body, &u)
	if err != nil {
		fmt.Println(err.Error())
	}

	if !loginUser(u) {
		fmt.Fprintf(w, "failed&Invalid username or password")
	} else {
		dirInfo, e := readDirectory(ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + u.Username)
		if e != nil {
			fmt.Fprintf(w, "failed&"+e.Error())
		} else {
			jsonDirInfo, err := json.Marshal(dirInfo)
			if err != nil {
				log.Print(err.Error())
			}
			fmt.Fprintf(w, "success&"+string(jsonDirInfo))
		}
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

	err = signupUser(newUser)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
	os.Mkdir(ROOT_DIR+string(os.PathSeparator)+USER_DIRECTORY_PREFIX+newUser.Username, 0777)
}

func delHandler(w http.ResponseWriter, r *http.Request) {
	fileName := strings.Replace(r.URL.String(), API_PREFIX+DELETE_PREFIX, "", 1)
	err := os.Remove(ROOT_DIR + string(os.PathSeparator) + fileName)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	} else {
		dirInfo, e := readDirectory(ROOT_DIR)
		if e != nil {
			fmt.Fprintf(w, e.Error())
		} else {
			jsonDirInfo, err := json.Marshal(dirInfo)
			if err != nil {
				log.Print(err.Error())
			} else {
				fmt.Fprintf(w, string(jsonDirInfo))
			}
		}
	}
	return
}

func uploadFileToServer(file multipart.File, header *multipart.FileHeader, username string) (err error) {
	userdir := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + username
	out, err := os.Create(userdir + string(os.PathSeparator) + header.Filename)
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
	http.ServeFile(w, r, ROOT_DIR+string(os.PathSeparator)+fileName)
}

func readDirectory(dirName string) (*dirInfo, error) {
	files, err := ioutil.ReadDir(dirName)
	var directoryInfo *dirInfo
	if err != nil {
		return directoryInfo, err
	}

	var fileInfos []fileInfo
	for _, finfo := range files {
		if !finfo.IsDir() {
			var fileInfo = new(fileInfo)
			fileInfo.Name = finfo.Name()
			fileInfo.Size = finfo.Size()
			fileInfos = append(fileInfos, *fileInfo)
		}
	}

	directoryInfo = new(dirInfo)
	directoryInfo.Files = fileInfos

	return directoryInfo, nil
}

func hasAccess(username string) bool {
	return USER_LOGIN_DATA[username]
}

func loginUser(u user) bool {
	user, err := getUser(u.Username)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	if user.Password == u.Password {
		USER_LOGIN_DATA[user.Username] = true
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
	err.userName = userName
	return u, err
}

func signupUser(newUser user) error {
	_, e := getUser(newUser.Username)
	if e == nil {
		var err = new(userExistsError)
		err.userName = newUser.Username
		return err
	}
	configFile := loadConfiguration()
	newUsersArray := append(USERS, newUser)
	configFile.Users = newUsersArray

	jsonConfiguration, err := json.Marshal(configFile)
	if err != nil {
		log.Fatal(err.Error())
	}
	ioutil.WriteFile("config.json", jsonConfiguration, 0777)

	USERS = newUsersArray
	USER_LOGIN_DATA[newUser.Username] = false
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

func initLoginData(users []user) {
	for _, user := range users {
		USER_LOGIN_DATA[user.Username] = false
	}
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
	initLoginData(USERS)
	applyConfiguration(configuration)

	// host client
	http.Handle("/", http.FileServer(http.Dir(".."+string(os.PathSeparator)+"client")))
	// handle file requests
	http.Handle(API_PREFIX, initServer())
	//start listening
	http.ListenAndServe(":"+strconv.Itoa(PORT), nil)
}
