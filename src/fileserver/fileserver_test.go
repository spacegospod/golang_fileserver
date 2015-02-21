package main

import (
	"testing"
	"time"
)

func TestGetStudent(t *testing.T) {
	// create test user
	user1 := new(user)
	user1.Password = "123"
	user1.Username = "user1"

	user2 := new(user)
	user2.Password = "123"
	user2.Username = "user2"

	// simulate application global variable
	USERS = []user{*user1, *user2}

	var u user
	var err error
	// test error
	u, err = getUser("nonExistingUser")
	if err == nil {
		t.Errorf("Got result for non existing user!")
	}

	// test username
	u, err = getUser("user1")
	if err != nil {
		t.Errorf("Failed to get user, got: %s!", err.Error())
	} else {
		if u.Username != "user1" {
			t.Errorf("Got wrong username! Expected 'user', got %s.", u.Username)
		}
	}

	// test password
	u, err = getUser("user2")
	if err != nil {
		t.Errorf("Failed to get user, got: %s!", err.Error())
	} else {
		if u.Password != "123" {
			t.Errorf("Got wrong password! Expected '123', got %s.", u.Password)
		}
	}
}

func TestLoginUser(t *testing.T) {
	// simulate application global variables
	user1 := new(user)
	user1.Password = "123"
	user1.Username = "user1"

	user2 := new(user)
	user2.Password = "123"
	user2.Username = "user2"

	user3 := new(user)
	user3.Password = "123"
	user3.Username = "user3"

	USERS = []user{*user1, *user2, *user3}

	// user2 is already logged in
	ld := new(loginData)
	ld.LoggedIn = true
	ld.LoginTime = time.Now()

	USER_LOGIN_DATA[user2.Username] = ld

	var success bool
	var msg = ""
	success, msg = loginUser(user1, "127.0.0.101:16567")
	if !success {
		t.Errorf("Failed to login user! %s", msg)
	}

	success, msg = loginUser(user2, "127.0.0.102:16567")
	if ACTIVE_ADDRESSES["127.0.0.102:16567"] != user2.Username {
		t.Errorf("Failed to track login IP address for user %s!", user2.Username)
	}

	success, msg = loginUser(user3, "127.0.0.103:16567")
	if !USER_LOGIN_DATA[user3.Username].LoggedIn {
		t.Errorf("Failed to initialize login data for user %s!", user3.Username)
	}
}

func TestLogoutUser(t *testing.T) {
	user1 := new(user)
	user1.Password = "123"
	user1.Username = "user1"
	loginUser(user1, "127.0.0.1:8080")
	logoutUser(user1)

	if ACTIVE_ADDRESSES["127.0.0.1:8080"] != "" {
		t.Errorf("Failed to clear login IP address data for user %s!", user1.Username)
	}

	if USER_LOGIN_DATA[user1.Username] != nil {
		t.Errorf("Failed to clear login data for user %s!", user1.Username)
	}
}

func TestHasAccess(t *testing.T) {
	// simulate application global variables
	user1 := new(user)
	user1.Password = "123"
	user1.Username = "user1"

	user2 := new(user)
	user2.Password = "123"
	user2.Username = "user2"

	user3 := new(user)
	user3.Password = "123"
	user3.Username = "user3"

	USERS = []user{*user1, *user2, *user3}

	var success bool
	var msg = ""

	success, _ = hasAccess(user1.Username, "127.0.0.101:16567")
	if success {
		t.Errorf("Unauthorized access for user %s", user1.Username)
	}

	loginUser(user2, "127.0.0.102:16567")
	success, msg = hasAccess(user2.Username, "127.0.0.102:16567")
	if !success {
		t.Errorf("Failed to get access permission for active user, got: %s", msg)
	}

	loginUser(user3, "127.0.0.103:16567")
	success, msg = hasAccess(user3.Username, "127.0.0.255:16567")
	if success {
		t.Errorf("Unauthorized access allowed for unknown address %s!", "127.0.0.255:16567")
	}
}
