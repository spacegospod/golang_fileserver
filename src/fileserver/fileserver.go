package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/pat"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	API_PREFIX        = "/api/"
	POST_PREFIX       = "post/"
	DELETE_PREFIX     = "delete/"
	DOWNLOAD_PREFIX   = "download/"
	UPLOAD_PREFIX     = "upload/"
	CREATE_DIR_PREFIX = "createdir/"
	NAVIGATION_PREFIX = "navigation/"
	LOGIN_PREFIX      = "login/"
	SIGNUP_PREFIX     = "signup/"

	USER_DIRECTORY_PREFIX = "user-"
	PORT                  = 8080
)

var (
	ROOT_DIR        string = "root"
	NAVIGATION_FWD         = "fwd"
	NAVIGATION_BACK        = "back"
	USER_LOGIN_DATA        = make(map[string]bool)
	USERS           []user
	USER_DIRS       = make(map[string]string)
)

/* --- types --- */
type userNotFoundError struct {
	username string
}

func (err *userNotFoundError) Error() string {
	return err.username + " does not exist"
}

type userExistsError struct {
	username string
}

func (err *userExistsError) Error() string {
	return "User " + err.username + " is already registered!"
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

type folderInfo struct {
	Name string `json:name`
}

type dirInfo struct {
	Files   []fileInfo   `json:files`
	Folders []folderInfo `json:folders`
}

/* --- HTTP handlers --- */
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	parameters := strings.Split(strings.Replace(r.URL.String(), API_PREFIX+DOWNLOAD_PREFIX, "", 1), "&")
	username := strings.Split(parameters[0], "=")[1]
	fileName := parameters[1]
	if hasAccess(username) {
		filePath := USER_DIRS[username] + string(os.PathSeparator) + fileName
		serveFileForDownload(w, r, filePath)
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
			dirPath := USER_DIRS[username]
			jsonDirInfo, e := generateDirectoryInfo(dirPath)

			if e != nil {
				fmt.Fprintf(w, e.Error())
			} else {
				fmt.Fprintf(w, jsonDirInfo)
			}
		}
	}
	return
}

func createDirHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.Replace(r.URL.String(), API_PREFIX+POST_PREFIX+CREATE_DIR_PREFIX, "", 1)
	if !hasAccess(username) {
		return
	}
	dirNameBytes, _ := ioutil.ReadAll(r.Body)
	dirName := string(dirNameBytes)

	err := os.Mkdir(ROOT_DIR+string(os.PathSeparator)+USER_DIRECTORY_PREFIX+username+string(os.PathSeparator)+dirName, 0777)

	if err != nil {
		fmt.Fprintf(w, err.Error())
	} else {
		currentPath := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + username
		jsonDirInfo, e := generateDirectoryInfo(currentPath)

		if e != nil {
			fmt.Fprintf(w, e.Error())
		} else {
			fmt.Fprintf(w, jsonDirInfo)
		}
	}

	return
}

func navigationHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: check params for consistency
	params := strings.Split(strings.Replace(r.URL.String(), API_PREFIX+POST_PREFIX+NAVIGATION_PREFIX, "", 1), "&")
	direction := params[0]
	username := params[1]

	if !hasAccess(username) {
		// TODO: err msg
		return
	}

	if direction == NAVIGATION_BACK {
		dirinfo, err := navigateBack(username)
		if err != nil {
			fmt.Fprintf(w, err.Error())
		} else {
			fmt.Fprintf(w, dirinfo)
		}
	} else if direction == NAVIGATION_FWD {
		dirname := params[2]

		dirinfo, err := openDir(username, dirname)
		if err != nil {
			fmt.Fprintf(w, err.Error())
		} else {
			fmt.Fprintf(w, dirinfo)
		}
	} else {
		// something's wrong
	}
}

func navigateBack(username string) (string, error) {
	currentDirPath := USER_DIRS[username]
	pathEntries := strings.Split(currentDirPath, string(os.PathSeparator))
	newDirPath := strings.Replace(currentDirPath, string(os.PathSeparator)+pathEntries[len(pathEntries)-1], "", 1)

	jsonDirInfo, e := generateDirectoryInfo(newDirPath)

	return jsonDirInfo, e
}

func openDir(username string, dirname string) (string, error) {
	dirPath := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + username + string(os.PathSeparator) + dirname
	USER_DIRS[username] = dirPath
	jsonDirInfo, e := generateDirectoryInfo(dirPath)

	return jsonDirInfo, e
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
		dirName := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + u.Username
		jsonDirInfo, e := generateDirectoryInfo(dirName)

		if e != nil {
			fmt.Fprintf(w, e.Error())
		} else {
			fmt.Fprintf(w, "success&"+jsonDirInfo)
		}
	}

	return
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

	return
}

func delHandler(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(strings.Replace(r.URL.String(), API_PREFIX+DELETE_PREFIX, "", 1), "&")
	username := params[0]
	itemName := params[1]

	path := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + username + string(os.PathSeparator) + itemName
	err := os.RemoveAll(path)

	if err != nil {
		fmt.Println(err.Error())
		fmt.Fprintf(w, err.Error())
	} else {
		dirName := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + username
		jsonDirInfo, e := generateDirectoryInfo(dirName)

		if e != nil {
			fmt.Fprintf(w, e.Error())
		} else {
			fmt.Fprintf(w, jsonDirInfo)
		}
	}

	return
}

func uploadFileToServer(file multipart.File, header *multipart.FileHeader, username string) (err error) {
	dir := USER_DIRS[username]
	out, err := os.Create(dir + string(os.PathSeparator) + header.Filename)
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

func serveFileForDownload(w http.ResponseWriter, r *http.Request, filepath string) {
	http.ServeFile(w, r, filepath)
	return
}

func generateDirectoryInfo(dirName string) (string, error) {
	dirInfo, e := readDirectory(dirName)
	if e != nil {
		log.Print(e.Error())
		return "", e
	} else {
		jsonDirInfo, err := json.Marshal(dirInfo)
		if err != nil {
			log.Print(err.Error())
			return "", e
		} else {
			return string(jsonDirInfo), nil
		}
	}
}

func readDirectory(dirName string) (*dirInfo, error) {
	items, err := ioutil.ReadDir(dirName)
	var directoryInfo *dirInfo
	if err != nil {
		return directoryInfo, err
	}

	var fileInfos []fileInfo
	var folderInfos []folderInfo
	for _, item := range items {
		if !item.IsDir() {
			var fileInfo = new(fileInfo)
			fileInfo.Name = item.Name()
			fileInfo.Size = item.Size()
			fileInfos = append(fileInfos, *fileInfo)
		} else {
			var folderInfo = new(folderInfo)
			folderInfo.Name = item.Name()
			folderInfos = append(folderInfos, *folderInfo)
		}
	}

	directoryInfo = new(dirInfo)
	directoryInfo.Files = fileInfos
	directoryInfo.Folders = folderInfos

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
		USER_DIRS[user.Username] = ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + user.Username
		return true
	}

	return false
}

func getUser(username string) (user, error) {
	var u user
	for _, user := range USERS {
		if user.Username == username {
			u = user
			return u, nil
		}
	}
	var err = new(userNotFoundError)
	err.username = username
	return u, err
}

func signupUser(newUser user) error {
	_, e := getUser(newUser.Username)
	if e == nil {
		var err = new(userExistsError)
		err.username = newUser.Username
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
	secondaryHandler.Post(API_PREFIX+POST_PREFIX+CREATE_DIR_PREFIX, http.HandlerFunc(createDirHandler))
	secondaryHandler.Post(API_PREFIX+POST_PREFIX+NAVIGATION_PREFIX, http.HandlerFunc(navigationHandler))
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
