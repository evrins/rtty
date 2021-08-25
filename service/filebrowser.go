package service

import (
	"github.com/asdine/storm"
	"github.com/filebrowser/filebrowser/v2/auth"
	"github.com/filebrowser/filebrowser/v2/diskcache"
	fbhttp "github.com/filebrowser/filebrowser/v2/http"
	"github.com/filebrowser/filebrowser/v2/img"
	"github.com/filebrowser/filebrowser/v2/settings"
	"github.com/filebrowser/filebrowser/v2/storage/bolt"
	"github.com/filebrowser/filebrowser/v2/users"
	"github.com/skanehira/rtty/fbdist"
	"github.com/spf13/afero"

	"log"
	"net/http"
	"os"
	"path/filepath"
)

func initFileBrowser() (fbHandler http.Handler, err error) {
	imgSrv := img.New(4)

	fbDir := filepath.Join(os.TempDir(), "fb")
	log.Println("filebrowser dir", fbDir)
	cacheDir := filepath.Join(fbDir, "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil { //nolint:govet,gomnd
		log.Fatalf("can't make directory %s: %s", cacheDir, err)
	}
	fileCache := diskcache.New(afero.NewOsFs(), cacheDir)

	storePath := filepath.Join(fbDir, "store.db")
	db, err := storm.Open(storePath)
	if err != nil {
		return
	}

	store, err := bolt.NewStorage(db)
	if err != nil {
		return
	}
	key, err := settings.GenerateKey()
	if err != nil {
		return
	}
	s := &settings.Settings{
		Key:           key,
		Signup:        false,
		CreateUserDir: false,
		Defaults: settings.UserDefaults{
			Scope:       ".",
			Locale:      "en",
			SingleClick: false,
			Perm: users.Permissions{
				Admin:    false,
				Execute:  true,
				Create:   true,
				Rename:   true,
				Modify:   true,
				Delete:   true,
				Share:    true,
				Download: true,
			},
		},
		AuthMethod: auth.MethodNoAuth,
	}

	err = store.Auth.Save(&auth.NoAuth{})
	if err != nil {
		return
	}

	err = store.Settings.Save(s)
	if err != nil {
		return
	}

	srvParams := &settings.Server{
		Root:             "/",
		BaseURL:          "/fb/",
		Log:              filepath.Join(fbDir, "log.log"),
		EnableThumbnails: true,
		ResizePreview:    true,
	}
	err = store.Settings.SaveServer(srvParams)
	if err != nil {
		return
	}


	password, _ := users.HashPwd("admin")

	user := &users.User{
		Username:     "admin",
		Password:     password,
		LockPassword: false,
	}

	s.Defaults.Apply(user)
	user.Perm.Admin = true

	err = store.Users.Save(user)

	fbHandler, err = fbhttp.NewHandler(imgSrv, fileCache, store, srvParams, fbdist.Dist)
	if err != nil {
		return
	}
	return
}
