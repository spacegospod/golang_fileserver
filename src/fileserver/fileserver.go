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
	"time"
)

const (
	API_PREFIX        = "/api/"
	GET_PREFIX        = "get/"
	POST_PREFIX       = "post/"
	DELETE_PREFIX     = "delete/"
	DOWNLOAD_PREFIX   = "download/"
	HOMEPAGE_PREFIX   = "home/"
	UPLOAD_PREFIX     = "upload/"
	CREATE_DIR_PREFIX = "createdir/"
	NAVIGATION_PREFIX = "navigation/"
	LOGIN_PREFIX      = "login/"
	SIGNUP_PREFIX     = "signup/"

	USER_DIRECTORY_PREFIX = "user-"
	PORT                  = 8080
)

var (
	ROOT_DIR         string = "root"
	SESSION_TIME     int64  = 0
	NAVIGATION_FWD          = "fwd"
	NAVIGATION_BACK         = "back"
	USER_LOGIN_DATA         = make(map[string]*loginData)
	ACTIVE_ADDRESSES        = make(map[string]string)
	USERS            []user
	USER_DIRS        = make(map[string]string)
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

type loginData struct {
	LoggedIn  bool
	LoginTime time.Time
}

type configFile struct {
	Config configuration `json:"config"`
	Users  []user        `json:"users"`
}

type configuration struct {
	RootDir     string `json:"rootDir"`
	SessionTime int64  `json:"sessionTime"`
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

/*
	Handler for GET requests on /api/get/download/
*/
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(strings.Replace(r.URL.String(), API_PREFIX+GET_PREFIX+DOWNLOAD_PREFIX, "", 1), "&")
	if len(params) < 2 {
		fmt.Fprintln(w, "Invalid URL parameters!")
		return
	}
	username := params[0]
	fileName := params[1]

	if acc, msg := hasAccess(username, r.RemoteAddr); !acc {
		fmt.Fprintln(w, msg)
		return
	}

	filePath := USER_DIRS[username] + string(os.PathSeparator) + fileName
	serveFileForDownload(w, r, filePath)
}

/*
	Handler for POST requests on /api/post/upload/
*/
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.Replace(r.URL.String(), API_PREFIX+POST_PREFIX+UPLOAD_PREFIX, "", 1)

	if len(username) == 0 {
		fmt.Fprintln(w, "Invalid URL parameters!")
		return
	}

	if acc, msg := hasAccess(username, r.RemoteAddr); !acc {
		fmt.Fprintln(w, msg)
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
}

/*
	Handler for POST requests on /api/post/createdir/
*/
func createDirHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.Replace(r.URL.String(), API_PREFIX+POST_PREFIX+CREATE_DIR_PREFIX, "", 1)

	if len(username) == 0 {
		fmt.Fprintln(w, "Invalid URL parameters!")
		return
	}

	if acc, msg := hasAccess(username, r.RemoteAddr); !acc {
		fmt.Fprintln(w, msg)
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
}

/*
	Handler for POST requests on /api/post/navigation/
*/
func navigationHandler(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(strings.Replace(r.URL.String(), API_PREFIX+POST_PREFIX+NAVIGATION_PREFIX, "", 1), "&")
	if len(params) < 3 {
		fmt.Fprintln(w, "Invalid URL parameters!")
		return
	}

	direction := params[0]
	username := params[1]

	if acc, msg := hasAccess(username, r.RemoteAddr); !acc {
		fmt.Fprintln(w, msg)
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
		fmt.Fprintln(w, "Invalid navigation!")
	}
}

/*
	Handler for GET requests on /api/get/home/
*/
func loadHomePageHandler(w http.ResponseWriter, r *http.Request) {
	username := ACTIVE_ADDRESSES[r.RemoteAddr]

	dirPath := USER_DIRS[username]
	jsonDirInfo, e := generateDirectoryInfo(dirPath)

	if e == nil {
		fmt.Fprintf(w, username+"&"+jsonDirInfo)
	}
}

/*
	Handler for POST requests on /api/post/login/
*/
func loginHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	var u user

	err := json.Unmarshal(body, &u)
	if err != nil {
		log.Println(err.Error())
		return
	}

	if success, msg := loginUser(&u, r.RemoteAddr); !success {
		fmt.Fprintf(w, "failed&"+msg)
	} else {
		dirName := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + u.Username
		jsonDirInfo, e := generateDirectoryInfo(dirName)

		if e != nil {
			log.Println(e.Error())
			fmt.Fprintf(w, "Cannot find user directory. Perhaps the root server directory has been moved?")
		} else {
			fmt.Fprintf(w, "success&"+jsonDirInfo)
		}
	}
}

/*
	Handler for POST requests on /api/post/signup/
*/
func signupHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	var newUser user

	err := json.Unmarshal(body, &newUser)
	if err != nil {
		log.Println(err.Error())
		return
	}

	err = signupUser(newUser)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	err = createUserDirectory(newUser.Username)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
}

/*
	Handler for DELETE requests on /api/delete/
*/
func delHandler(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(strings.Replace(r.URL.String(), API_PREFIX+DELETE_PREFIX, "", 1), "&")
	username := params[0]
	itemName := params[1]

	if acc, msg := hasAccess(username, r.RemoteAddr); !acc {
		fmt.Fprintln(w, msg)
		return
	}

	path := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + username + string(os.PathSeparator) + itemName
	err := os.RemoveAll(path)

	if err != nil {
		log.Println(err.Error())
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
}

/*
	Retrieves folder data for the parent directory of the provided user's currently active one
*/
func navigateBack(username string) (string, error) {
	currentDirPath := USER_DIRS[username]
	if len(currentDirPath) == 0 {
		return "", new(userNotFoundError)
	}
	pathEntries := strings.Split(currentDirPath, string(os.PathSeparator))
	newDirPath := strings.Replace(currentDirPath, string(os.PathSeparator)+pathEntries[len(pathEntries)-1], "", 1)

	jsonDirInfo, e := generateDirectoryInfo(newDirPath)
	if e == nil {
		USER_DIRS[username] = newDirPath
	}

	return jsonDirInfo, e
}

/*
	Retrieves folder data for the child directory with the provided name (dirname)
*/
func openDir(username string, dirname string) (string, error) {
	dirPath := ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + username + string(os.PathSeparator) + dirname
	jsonDirInfo, e := generateDirectoryInfo(dirPath)
	if e == nil {
		USER_DIRS[username] = dirPath
	}

	return jsonDirInfo, e
}

/*
	Adds the new file to the filesystem.
*/
func uploadFileToServer(file multipart.File, header *multipart.FileHeader, username string) (err error) {
	dir := USER_DIRS[username]
	out, err := os.Create(dir + string(os.PathSeparator) + header.Filename)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

/*
	A horse walks into a bar..
*/
func serveFileForDownload(w http.ResponseWriter, r *http.Request, filepath string) {
	http.ServeFile(w, r, filepath)
	return
}

/*
	Creates a directory on the filesystem for the provided user in the following format: user-<username>
*/
func createUserDirectory(username string) error {
	err := os.Mkdir(ROOT_DIR+string(os.PathSeparator)+USER_DIRECTORY_PREFIX+username, 0777)
	return err
}

/*
	Generates a string representation(json) of the file structure in the specified directory.
	The result from this method contains file/directory names and file sizes.
*/
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

/*
	Returns a pointer to a dirInfo object containing the data for the specified
	folder (file/directory names and file sizes)
*/
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

/*
	Validates whether the provided user has access to the server.

	The user can only access the server from the IP addresses from which
	a login occured.
	If the session time (see config file) for the provided user has expired
	automatic logout gets triggered.
*/
func hasAccess(username string, addr string) (bool, string) {
	if username == "" || addr == "" {
		return false, "Access denied!"
	}

	loginData := USER_LOGIN_DATA[username]

	if loginData != nil && loginData.LoggedIn {
		if int64(time.Since(loginData.LoginTime).Seconds()) > SESSION_TIME {
			user, _ := getUser(username)
			logoutUser(&user)
			return false, "Session timed out!"
		}

		if ACTIVE_ADDRESSES[addr] == username {
			return true, ""
		}
	}

	return false, "Access denied!"
}

/*
	Checks whether the provided credentials are valid (present in the list of
	registered users, see config file). If so, performs a login,
	which consists of setting a timestamp and recording the IP address.
	Also sets the active directory for the provided user to its root value (user-<username>)
*/
func loginUser(u *user, addr string) (bool, string) {
	user, err := getUser(u.Username)
	if err != nil {
		log.Println(err.Error())
		return false, err.Error()
	}

	if user.Password == u.Password {
		ld := USER_LOGIN_DATA[user.Username]
		if ld == nil {
			ld = new(loginData)
			ld.LoggedIn = true
			ld.LoginTime = time.Now()

			USER_LOGIN_DATA[user.Username] = ld
		}

		ACTIVE_ADDRESSES[addr] = user.Username

		USER_DIRS[user.Username] = ROOT_DIR + string(os.PathSeparator) + USER_DIRECTORY_PREFIX + user.Username
		return true, ""
	}

	return false, "Invalid username or password"
}

/*
	Clears all login data stored for the provided user
*/
func logoutUser(u *user) {
	USER_LOGIN_DATA[u.Username] = nil
	for key, _ := range ACTIVE_ADDRESSES {
		if ACTIVE_ADDRESSES[key] == u.Username {
			delete(ACTIVE_ADDRESSES, key)
		}
	}
}

/*
	Retrieves the 'user' object for the provided username.
	If the user is not present returns a 'userNotFoundError'
*/
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

/*
	Performs a signup for the provided user. The operation updates the server
	configuration file with the new user's data.
*/
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
	return nil
}

/*
	Reads the server configuration file and emits a 'configFile' object.
*/
func loadConfiguration() configFile {
	path, _ := os.Getwd()
	file, err := os.Open(path + string(os.PathSeparator) + "config.json")
	defer file.Close()
	if err != nil {
		log.Println(err.Error())
		log.Fatal("Failed to load configuration!")
	}

	var buffer bytes.Buffer
	io.Copy(&buffer, file)

	var configFile configFile
	// read config file
	err = json.Unmarshal(buffer.Bytes(), &configFile)
	if err != nil {
		log.Println(err.Error())
		log.Fatal("Configuration file is corrupt!")
	}
	return configFile
}

/*
	Sets server runtime parameters based on the provided configuration.
*/
func applyConfiguration(config configuration) {
	ROOT_DIR = config.RootDir
	SESSION_TIME = config.SessionTime
	if _, err := os.Stat(ROOT_DIR); os.IsNotExist(err) {
		err = os.Mkdir(ROOT_DIR, 0777)
		if err != nil {
			log.Println(err.Error())
		}
	}
}

/*
	Initializes the server with all HTTP handlers.
*/
func initServer() http.Handler {
	server := pat.New()

	getServer := pat.New()
	getServer.Get(API_PREFIX+GET_PREFIX+DOWNLOAD_PREFIX, http.HandlerFunc(downloadHandler))
	getServer.Get(API_PREFIX+GET_PREFIX+HOMEPAGE_PREFIX, http.HandlerFunc(loadHomePageHandler))

	postServer := pat.New()
	postServer.Post(API_PREFIX+POST_PREFIX+UPLOAD_PREFIX, http.HandlerFunc(uploadHandler))
	postServer.Post(API_PREFIX+POST_PREFIX+CREATE_DIR_PREFIX, http.HandlerFunc(createDirHandler))
	postServer.Post(API_PREFIX+POST_PREFIX+NAVIGATION_PREFIX, http.HandlerFunc(navigationHandler))
	postServer.Post(API_PREFIX+POST_PREFIX+LOGIN_PREFIX, http.HandlerFunc(loginHandler))
	postServer.Post(API_PREFIX+POST_PREFIX+SIGNUP_PREFIX, http.HandlerFunc(signupHandler))

	server.Get(API_PREFIX+GET_PREFIX, getServer)
	server.Post(API_PREFIX+POST_PREFIX, postServer)
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
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(PORT), nil))
}
