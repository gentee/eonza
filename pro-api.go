package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"eonza/lib"
	"eonza/users"

	"github.com/gentee/gentee/vm"
	echo "github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type SettingsResponse struct {
	Settings users.ProSettings `json:"settings"`
	Error    string            `json:"error,omitempty"`
}

type MasterPass struct {
	Master     string `json:"master"`
	ConfMaster string `json:"confmaster,omitempty"`
	Current    string `json:"current,omitempty"`
}

type RolesResponse struct {
	List  []users.Role `json:"list,omitempty"`
	Error string       `json:"error,omitempty"`
}

type UsersResponse struct {
	List  []users.User `json:"list,omitempty"`
	Error string       `json:"error,omitempty"`
}

type StorageItem struct {
	Secure
	Name string `json:"name"`
}

type StorageResponse struct {
	Encrypted bool          `json:"encrypted"`
	Created   bool          `json:"created"`
	List      []StorageItem `json:"list"`
	Error     string        `json:"error,omitempty"`
}

type License struct {
	License string `json:"license"`
}

type SaveForm struct {
	UserID uint32                 `json:"userid"`
	TaskID uint32                 `json:"taskid"`
	Ref    string                 `json:"ref"`
	Form   map[string]interface{} `json:"form"`
}

type AutoFill struct {
	UserID uint32 `json:"userid"`
	TaskID uint32 `json:"taskid"`
	Ref    string `json:"ref"`
}

type AutoFillResponse struct {
	AutoFill []map[string]interface{} `json:"autofill"`
	Error    string                   `json:"error,omitempty"`
}

var (
	ErrProDisabled = echo.NewHTTPError(http.StatusForbidden, "Pro version is disabled")
	ErrAccess      = echo.NewHTTPError(http.StatusForbidden, "Access denied")
	ErrEmptyName   = fmt.Errorf(`Name is empty`)
)

func AdminAccess(userID uint32) error {
	if userID == users.XRootID {
		return nil
	}
	if user, ok := GetUser(userID); !ok || user.RoleID != users.XAdminID {
		return ErrAccess
	}
	return nil
}

func ProAccess(userID uint32) error {
	if !Active {
		return ErrProDisabled
	}
	return AdminAccess(userID)
}

func ScriptAccess(name, ipath string, roleid uint32) error {
	if roleid == users.XAdminID || roleid >= users.ResRoleID {
		return nil
	}
	if role, ok := GetRole(roleid); ok && users.MatchAllow(name, ipath, role) {
		return nil
	}
	return ErrAccess
}

/*
func Access(userID uint32) (users.Role, error) {
	if userID == users.XRootID {
		return users.XAdminID, nil
	}
	user, err := GetUser(userID)
	if err != nil {
		return 0, err
	}
	return user.Role, nil
}*/

func rolesResponse(c echo.Context) error {
	list := make([]users.Role, 0)
	for _, role := range proStorage.Roles {
		if role.ID >= users.ResRoleID {
			continue
		}
		list = append(list, role)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return c.JSON(http.StatusOK, &RolesResponse{
		List: list,
	})
}

func usersResponse(c echo.Context) error {
	list := make([]users.User, 0)
	for _, user := range proStorage.Users {
		list = append(list, user)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Nickname < list[j].Nickname
	})
	return c.JSON(http.StatusOK, &UsersResponse{
		List: list,
	})
}

func rolesHandle(c echo.Context) error {
	if err := AdminAccess(c.(*Auth).User.ID); err != nil {
		return err
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	return rolesResponse(c)
}

func usersHandle(c echo.Context) error {
	if err := AdminAccess(c.(*Auth).User.ID); err != nil {
		return err
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	return usersResponse(c)
}

func logoutAll() error {
	for _, user := range proStorage.Users {
		user.PassCounter++
		proStorage.Users[user.ID] = user
	}
	if err := ProSaveStorage(false); err != nil {
		return err
	}
	if err := CallbackPassCounter(); err != nil {
		return err
	}
	return nil
}

func logoutallHandle(c echo.Context) error {
	errResult := func(err error) error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult(err)
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	if err := logoutAll(); err != nil {
		return errResult(err)
	}
	return c.JSON(http.StatusOK, Response{})
}

func saveRoleHandle(c echo.Context) error {
	errResult := func(err error) error {
		return c.JSON(http.StatusOK, RolesResponse{Error: fmt.Sprint(err)})
	}
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult(err)
	}
	var (
		role users.Role
	)
	if err := c.Bind(&role); err != nil {
		return errResult(err)
	}
	if len(role.Name) == 0 {
		return errResult(ErrEmptyName)
	}
	if role.ID == users.XAdminID || role.ID >= users.ResRoleID {
		return errResult(ErrAccess)
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	if role.ID == 0 {
		for {
			role.ID = lib.RndNum()
			if _, ok := proStorage.Roles[role.ID]; !ok && role.ID < users.ResRoleID {
				break
			}
		}
	}
	for _, item := range proStorage.Roles {
		if strings.ToLower(role.Name) == strings.ToLower(item.Name) && role.ID != item.ID {
			return errResult(fmt.Errorf(`Name %s exists`, role.Name))
		}
	}
	proStorage.Roles[role.ID] = users.ParseAllow(role)
	if err := ProSaveStorage(false); err != nil {
		return errResult(err)
	}
	return rolesResponse(c)
}

func saveUserHandle(c echo.Context) error {
	errResult := func(err error) error {
		return c.JSON(http.StatusOK, RolesResponse{Error: fmt.Sprint(err)})
	}
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult(err)
	}
	var (
		user     users.User
		password string
	)
	if err := c.Bind(&user); err != nil {
		return errResult(err)
	}
	userp := strings.SplitN(user.Nickname, `/@/`, 2)
	if len(userp) > 1 {
		password = userp[1]
	}
	user.Nickname = userp[0]
	if len(user.Nickname) == 0 {
		return errResult(ErrEmptyName)
	}
	if user.ID == users.XRootID {
		return ErrAccess
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	if user.ID == 0 {
		if len(password) == 0 {
			return errResult(fmt.Errorf(`Empty password`))
		}
		for {
			user.ID = lib.RndNum()
			if _, ok := proStorage.Users[user.ID]; !ok {
				break
			}
		}
	} else if curuser, ok := proStorage.Users[user.ID]; !ok {
		return ErrAccess
	} else {
		user.PassCounter = curuser.PassCounter
		user.PasswordHash = curuser.PasswordHash
	}
	for _, item := range proStorage.Users {
		if strings.ToLower(user.Nickname) == strings.ToLower(item.Nickname) && user.ID != item.ID {
			return errResult(fmt.Errorf(`User '%s' exists`, user.Nickname))
		}
	}
	if len(password) > 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 11)
		if err != nil {
			return errResult(err)
		}
		user.PassCounter++
		user.PasswordHash = hash
	}
	proStorage.Users[user.ID] = user
	if err := ProSaveStorage(false); err != nil {
		return errResult(err)
	}
	return usersResponse(c)
}

func removeRoleHandle(c echo.Context) error {
	errResult := func(err error) error {
		return c.JSON(http.StatusOK, RolesResponse{Error: fmt.Sprint(err)})
	}
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult(err)
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if id == users.XAdminID || id >= users.ResRoleID {
		return ErrAccess
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	for _, user := range proStorage.Users {
		if user.RoleID == uint32(id) {
			return errResult(fmt.Errorf(`There is a user with this role`))
		}
	}
	if _, ok := proStorage.Roles[uint32(id)]; ok {
		delete(proStorage.Roles, uint32(id))
		if err := ProSaveStorage(false); err != nil {
			return errResult(err)
		}
	}
	return rolesResponse(c)
}

func removeUserHandle(c echo.Context) error {
	errResult := func(err error) error {
		return c.JSON(http.StatusOK, UsersResponse{Error: fmt.Sprint(err)})
	}
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult(err)
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if id == users.XRootID {
		return ErrAccess
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	if _, ok := proStorage.Users[uint32(id)]; ok {
		delete(proStorage.Users, uint32(id))
		DeleteUser(uint32(id))
		if err := ProSaveStorage(false); err != nil {
			return errResult(err)
		}
	}
	return usersResponse(c)
}

func reset2faHandle(c echo.Context) error {
	errResult := func(err error) error {
		return c.JSON(http.StatusOK, UsersResponse{Error: fmt.Sprint(err)})
	}
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult(err)
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	proMutex.Lock()
	defer proMutex.Unlock()
	uid := uint32(id)
	if _, ok := proStorage.Twofa[uid]; ok {
		if user, ok := proStorage.Users[uid]; ok {
			user.PassCounter++
			proStorage.Users[uid] = user
		}
		delete(proStorage.Twofa, uid)
		if err := ProSaveStorage(false); err != nil {
			return errResult(err)
		}
		if uid == users.XRootID {
			if err := CallbackPassCounter(); err != nil {
				return errResult(err)
			}
		}
	}
	return c.JSON(http.StatusOK, &Response{})
}

func saveProSetHandle(c echo.Context) error {
	var err error
	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult()
	}
	twofa := proStorage.Settings.Twofa
	if err = c.Bind(&proStorage.Settings); err != nil {
		return errResult()
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	if !twofa && proStorage.Settings.Twofa {
		if err = logoutAll(); err != nil {
			return errResult()
		}
	}
	if err = ProSaveStorage(false); err != nil {
		return errResult()
	}
	return c.JSON(http.StatusOK, &SettingsResponse{Settings: proStorage.Settings})
}

func createStorageHandle(c echo.Context) error {
	var (
		err    error
		master MasterPass
	)
	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult()
	}
	if err = c.Bind(&master); err != nil {
		return errResult()
	}
	if len(master.Master) == 0 || master.Master != master.ConfMaster {
		err = fmt.Errorf(`Empty master password or wrong confrimation`)
		return errResult()
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	if len(proStorage.Settings.Master) != 0 {
		err = fmt.Errorf(`Storage already exists`)
		return errResult()
	}
	passphrase = []byte(master.Master)
	shaHash := sha256.Sum256(passphrase)
	proStorage.Settings.Master = strings.ToLower(hex.EncodeToString(shaHash[:]))
	secure = make(map[string]Secure)
	secureConst = make(map[string]string)
	if err = ProSaveStorage(false); err != nil {
		return errResult()
	}
	return respStorageHandle(c)
}

func PassStorage() (response StorageResponse) {
	response.Created = len(proStorage.Settings.Master) > 0
	response.Encrypted = secure == nil
	list := make([]StorageItem, 0, len(secure))
	for key, item := range secure {
		item.Value = ``
		list = append(list, StorageItem{
			Secure: item,
			Name:   key,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	response.List = list
	return response
}

func respStorageHandle(c echo.Context) error {
	return c.JSON(http.StatusOK, PassStorage())
}

func storageHandle(c echo.Context) error {
	var (
		err      error
		response StorageResponse
	)
	errResult := func() error {
		response.Error = fmt.Sprint(err)
		return c.JSON(http.StatusOK, response)
	}
	if err = AdminAccess(c.(*Auth).User.ID); err != nil {
		return errResult()
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	return respStorageHandle(c)
}

func decryptStorageHandle(c echo.Context) error {
	var (
		err    error
		master MasterPass
		ret    []byte
	)
	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult()
	}
	if err = c.Bind(&master); err != nil {
		return errResult()
	}
	if len(master.Master) == 0 {
		err = fmt.Errorf(`Empty master password`)
		return errResult()
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	passphrase = []byte(master.Master)
	shaHash := sha256.Sum256(passphrase)
	if proStorage.Settings.Master != strings.ToLower(hex.EncodeToString(shaHash[:])) {
		err = fmt.Errorf(`Invalid password`)
		return errResult()
	}
	secure = make(map[string]Secure)
	secureConst = make(map[string]string)
	if len(proStorage.Secure) > 0 {
		if ret, err = vm.AESDecrypt([]byte(master.Master), proStorage.Secure); err != nil {
			return errResult()
		}
		if err = yaml.Unmarshal(ret, &secure); err != nil {
			return errResult()
		}
		for key, item := range secure {
			secureConst[key] = item.Value
		}
	}
	return respStorageHandle(c)
}

func encryptStorageHandle(c echo.Context) error {
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	secure = nil
	secureConst = nil
	passphrase = nil
	return respStorageHandle(c)
}

func saveStorageHandle(c echo.Context) error {
	errResult := func(err error) error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult(err)
	}
	var (
		sitem StorageItem
	)
	if err := c.Bind(&sitem); err != nil {
		return errResult(err)
	}
	if len(sitem.Name) == 0 {
		return errResult(ErrEmptyName)
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	var (
		cur   Secure
		todel string
	)
	if sitem.ID == 0 {
	main:
		for {
			sitem.ID = lib.RndNum()
			for _, item := range secure {
				if item.ID == sitem.ID {
					continue main
				}
			}
			break
		}
		cur = sitem.Secure
	} else {
		for key, item := range secure {
			if item.ID == sitem.ID {
				cur = item
				if key != sitem.Name {
					todel = key
				}
				cur.Desc = sitem.Desc
				if len(sitem.Value) > 0 {
					cur.Value = strings.TrimSpace(sitem.Value)
				}
				break
			}
		}
		if cur.ID == 0 {
			return errResult(fmt.Errorf(`Item %d doesn't exist`, sitem.ID))
		}
	}
	if icur, ok := secure[sitem.Name]; ok && sitem.ID != icur.ID {
		return errResult(fmt.Errorf(`Name %s exists`, sitem.Name))
	}
	if len(todel) != 0 {
		delete(secure, todel)
		delete(secureConst, todel)
	}
	secure[sitem.Name] = cur
	secureConst[sitem.Name] = cur.Value
	if err := ProSaveStorage(true); err != nil {
		return errResult(err)
	}
	return respStorageHandle(c)
}

func removeStorageHandle(c echo.Context) error {
	errResult := func(err error) error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err := ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult(err)
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	proMutex.Lock()
	defer proMutex.Unlock()
	var todel string
	for key, item := range secure {
		if item.ID == uint32(id) {
			todel = key
		}
	}
	if len(todel) > 0 {
		delete(secure, todel)
		delete(secureConst, todel)
		if err := ProSaveStorage(true); err != nil {
			return errResult(err)
		}
	}
	return respStorageHandle(c)
}

func changePasswordHandle(c echo.Context) error {
	var (
		err    error
		master MasterPass
	)
	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = ProAccess(c.(*Auth).User.ID); err != nil {
		return errResult()
	}
	if err = c.Bind(&master); err != nil {
		return errResult()
	}
	if len(master.Master) == 0 || master.Master != master.ConfMaster {
		err = fmt.Errorf(`Empty master password or wrong confrimation`)
		return errResult()
	}
	proMutex.Lock()
	defer proMutex.Unlock()
	if secure == nil {
		err = fmt.Errorf(`Storage is disabled`)
		return errResult()
	}
	shaHash := sha256.Sum256([]byte(master.Current))
	if proStorage.Settings.Master != strings.ToLower(hex.EncodeToString(shaHash[:])) {
		err = fmt.Errorf(`Invalid password`)
		return errResult()
	}
	passphrase = []byte(master.Master)
	shaHash = sha256.Sum256(passphrase)
	proStorage.Settings.Master = strings.ToLower(hex.EncodeToString(shaHash[:]))
	if err = ProSaveStorage(true); err != nil {
		return errResult()
	}
	return respStorageHandle(c)
}

func saveFormHandle(c echo.Context) error {
	var (
		status bool
		err    error
		form   SaveForm
	)
	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = c.Bind(&form); err != nil {
		return errResult()
	}
	if status, err = CallbackTaskCheck(form.TaskID, form.UserID); err != nil {
		if status {
			return errResult()
		} else {
			return c.JSON(http.StatusOK, Response{})
		}
	}
	if err = SetUserForms(form.UserID, form.Ref, form.Form); err != nil {
		return errResult()
	}
	return c.JSON(http.StatusOK, Response{})
}

func autoFillHandle(c echo.Context) error {
	var (
		status bool
		err    error
		auto   AutoFill
		resp   AutoFillResponse
	)
	errResult := func() error {
		return c.JSON(http.StatusOK, Response{Error: fmt.Sprint(err)})
	}
	if err = c.Bind(&auto); err != nil {
		return errResult()
	}
	if status, err = CallbackTaskCheck(auto.TaskID, auto.UserID); err != nil {
		if status {
			return errResult()
		} else {
			return c.JSON(http.StatusOK, resp)
		}
	}
	if user, ok := usersPro[c.(*Auth).User.ID]; ok {
		resp.AutoFill = user.Forms[auto.Ref]
	}
	return c.JSON(http.StatusOK, resp)
}

func ProApi(e *echo.Echo) {
	e.GET("/api/roles", rolesHandle)
	e.GET("/api/removerole/:id", removeRoleHandle)
	e.GET("/api/removeuser/:id", removeUserHandle)
	e.GET("/api/removestorage/:id", removeStorageHandle)
	e.GET("/api/users", usersHandle)
	e.GET("/api/logoutall", logoutallHandle)
	e.GET("/api/reset2fa/:id", reset2faHandle)
	e.GET("/api/storage", storageHandle)
	e.POST("/api/role", saveRoleHandle)
	e.POST("/api/user", saveUserHandle)
	e.POST("/api/proset", saveProSetHandle)
	e.POST("/api/createstorage", createStorageHandle)
	e.POST("/api/decryptstorage", decryptStorageHandle)
	e.POST("/api/encryptstorage", encryptStorageHandle)
	e.POST("/api/storage", saveStorageHandle)
	e.POST("/api/storagepassword", changePasswordHandle)
	e.POST("/api/autofill", autoFillHandle)
	e.POST("/api/saveform", saveFormHandle)
}
