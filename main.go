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
	"gorm.io/driver/sqlite"
	"time"
)

func generateToken() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex"`
	Password string

	SessionID string
	SessionExpiry time.Time

	RoleIDs []string `gorm:"type:text[]"`
}

type Role struct {
	gorm.Model

	ID string `gorm:"uniqueIndex"`

	Name string

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

	ChildrenIDs []string `gorm:"type:text[]"`
}

type Invite struct {
	gorm.Model
	Code string `gorm:"uniqueIndex"`
	Expiry time.Time
	RoleIDs []string `gorm:"type:text[]"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	addr := os.Getenv("HOST_ADDRESS")

	db, err := gorm.Open(sqlite.Open("./database.db"), &gorm.Config{})
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

	mux.HandleFunc("/api/invite", func(w http.ResponseWriter, r *http.Request) {
	})

	fmt.Println("Server started at " + addr)

	log.Fatal(httpServer.ListenAndServe())
}
