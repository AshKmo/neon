package main

import (
	"fmt"
	"net/http"
	"log"
	"os"
	"math/rand"
	"github.com/joho/godotenv"
	"encoding/hex"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/driver/sqlite"
	"time"
	"errors"
	"encoding/json"
	"database/sql/driver"
	"golang.org/x/crypto/bcrypt"
)

func generateToken() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type StringArray []string

func (arr StringArray) Value() (driver.Value, error) {
	return json.Marshal(arr)
}

func (arr *StringArray) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan StringArray json")
	}
	return json.Unmarshal(bytes, arr)
}

type User struct {
	gorm.Model

	Username string `gorm:"primaryKey"`
	Password string

	SessionToken string
	SessionExpiry int64

	RoleIDs StringArray `gorm:"type:text"`
}

type Role struct {
	gorm.Model

	Name string `gorm:"primaryKey"`

	CanCreateTemplates bool
	CanCreateItems bool

	CanCreateChildren bool
	CanEditChildren bool
	CanDeleteChildren bool
	CanMoveChildren bool

	CanInvite bool

	CanEditUser bool
	CanDeleteUser bool
	CanAddRolesToUser bool

	ChildrenIDs StringArray `gorm:"type:text"`
}

type Invite struct {
	gorm.Model

	Code string `gorm:"primaryKey"`
	Expiry int64
	RoleIDs StringArray `gorm:"type:text"`
}

func generateInvite(db *gorm.DB, timeLength int64, roleIDs []string) (string, error) {
	code, err := generateToken()

	invite := &Invite{
		Code: code,
		Expiry: time.Now().Unix() + timeLength,
		RoleIDs: roleIDs,
	}

	db.Create(invite)

	if err != nil {
		return "", err
	}

	return os.Getenv("ORIGIN") + "/invite.html?c=" + code, nil
}

type Interval struct {
	ticker *time.Ticker
	quit chan struct{}
}

func NewInterval(function func(), period time.Duration) *Interval {
	interval := &Interval{
		ticker: time.NewTicker(period * time.Second),
		quit: make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-interval.ticker.C:
				function()
			case <-interval.quit:
				interval.ticker.Stop()
			}
		}
	}()

	return interval
}

func httpError(w http.ResponseWriter, n int) {
	switch n {
	case 401:
		http.Error(w, "401 Unauthorized", n)
	case 405:
		http.Error(w, "405 Method Not Allowed", n)
	case 500:
		http.Error(w, "500 Internal Server Error", n)
	}
}

func auth(db *gorm.DB, r *http.Request) (User, error) {
	var user User

	token, err := r.Cookie("token")
	if err != nil {
		return user, errors.New("token not specified")
	}

	err = db.Where("session_token = ?", token.Value).First(&user).Error
	if err != nil {
		return user, errors.New("invalid token")
	}

	return user, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	databasePath := "./database.db"
	
	_, err = os.Stat(databasePath)
	firstTime := errors.Is(err, os.ErrNotExist)

	if firstTime {
		fmt.Println("No database detected; first-time setup will be run\n")
	}

	addr := os.Getenv("HOST_ADDRESS")

	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&User{}, &Role{}, &Invite{})

	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr: addr,
		Handler: mux,
	}

	mux.Handle("/", http.FileServer(http.Dir("./site")))

	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		_, err := auth(db, r)
		if err != nil {
			httpError(w, 401)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "Pong!")
	})

	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			httpError(w, 405)
			return
		}

		var user User

		err := db.Where("username = ?", r.FormValue("username")).First(&user).Error
		if err != nil {
			httpError(w, 401)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(r.FormValue("password")))
		if err != nil {
			httpError(w, 401)
			return
		}

		if user.SessionExpiry <= time.Now().Unix() {
			token, err := generateToken()
			if err != nil {
				httpError(w, 500)
				return
			}

			user.SessionToken = token
		}

		unixTimeNow := time.Now().Unix()

		// leave the user signed in for a week
		const loginTime int = 86400 * 7

		user.SessionExpiry = unixTimeNow + int64(loginTime)

		http.SetCookie(w, &http.Cookie{
			Name: "token",
			Value: user.SessionToken,
			MaxAge: loginTime,
		})

		db.Save(&user)

		http.Redirect(w, r, "/", 302)
	})

	mux.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}

		var invite Invite
		err := db.Where("code = ?", r.FormValue("invite")).First(&invite).Error
		if err != nil {
			httpError(w, 401)
			return
		}

		password := r.FormValue("password")

		hash, err := bcrypt.GenerateFromPassword([]byte(password), 8)
		if err != nil {
			httpError(w, 500)
			return
		}

		user := &User{
			Username: r.FormValue("username"),
			// TODO: proper hashing
			Password: string(hash),
			RoleIDs: invite.RoleIDs,
		}

		db.Create(user)

		db.Delete(&invite)
	})

	if firstTime {
		db.Create(&Role{
			Name: "root",
		})

		const rootInviteLength int64 = 300

		invite, err := generateInvite(db, rootInviteLength, []string{"root"})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Welcome to Neon! Here is your one-time invite link for full (root) permissions:")
		fmt.Println(invite)
		fmt.Println("This invite will self-destruct in", rootInviteLength / 60, "minutes\n")
	}

	// clean old invites
	NewInterval(func() {
		db.Where("expiry <= ?", time.Now().Unix()).Delete(&Invite{})
	}, 5)

	fmt.Println("Server listening on", addr, "at origin", os.Getenv("ORIGIN"))

	log.Fatal(httpServer.ListenAndServe())
}
