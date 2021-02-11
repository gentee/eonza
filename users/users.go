// Copyright 2021 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package users

const (
	RootUser = `root`
	RootID   = 0 // Don't change. Must be zero
	RootRole = `admin`
)

type Role struct {
	ID            uint32 `json:"id"`
	Name          string `json:"name"`
	Allow         string `json:"allow"`
	Tasks         int    `json:"tasks"`
	Notifications int    `json:"notifications"`
	//	Disallow string
	//	Scripts       int
	//	Settings      int
	//	Pro           int
}

type User struct {
	ID           uint32
	Nickname     string
	PasswordHash []byte
	Role         uint32
}

var (
	Users map[uint32]User
	Roles []Role
)

func InitRoot(psw []byte) {
	Roles = []Role{
		{ID: RootID, Name: RootRole},
	}
	Users = map[uint32]User{
		RootID: {ID: RootID, Nickname: RootUser, PasswordHash: psw, Role: RootID},
	}
}

/*
func NewUser(nickname string) (uint32, error) {
	user := User{
		Nickname: nickname,
	}
	if !lib.ValidateSysName(nickname) {
		return 0, fmt.Errorf(Lang(DefLang, `invalidfield`), Lang(DefLang, `nickname`))
	}
	for _, item := range storage.Users {
		if item.Nickname == nickname {
			return 0, fmt.Errorf(Lang(DefLang, `errnickname`), nickname)
		}

	}
	private, public, err := lib.GenerateKeys()
	if err != nil {
		return 0, err
	}
	user.PublicKey = public
	user.ID = crc32.ChecksumIEEE(private)
	if err = ioutil.WriteFile(filepath.Join(cfg.Users.Dir, user.Nickname+`.key`),
		[]byte(hex.EncodeToString(private)), 0777 os.ModePerm); err != nil {
		return 0, err
	}
	storage.Users[user.ID] = &user
	userSettings[user.ID] = UserSettings{
		ID:   user.ID,
		Lang: appInfo.Lang,
	}
	return user.ID, nil
}
*/
