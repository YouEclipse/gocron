package models

import (
    "gocron/modules/utils"
    "time"
)

const PasswordSaltLength = 6

// 用户model
type User struct {
    Id       int       `xorm:"pk autoincr notnull "`
    Name     string    `xorm:"varchar(32) notnull unique"`             // 用户名
    Password string    `xorm:"char(32) notnull "`                      // 密码
    Salt     string    `xorm:"char(6) notnull "`                       // 密码盐值
    Email    string    `xorm:"varchar(50) notnull unique default '' "` // 邮箱
    Created  time.Time `xorm:"datetime notnull created"`
    Updated  time.Time `xorm:"datetime updated"`
    Deleted  time.Time `xorm:"datetime deleted"`
    IsAdmin  int8      `xorm:"tinyint notnull default 0"` // 是否是管理员 1:管理员 0:普通用户
    Status   Status    `xorm:"tinyint notnull default 1"` // 1: 正常 0:禁用
    BaseModel      `xorm:"-"`
}

// 新增
func (user *User) Create() (insertId int, err error) {
    user.Status = Enabled
    user.Salt = user.generateSalt()
    user.Password = user.encryptPassword(user.Password, user.Salt)

    _, err = Db.Insert(user)
    if err == nil {
        insertId = user.Id
    }

    return
}

// 更新
func (user *User) Update(id int, data CommonMap) (int64, error) {
    return Db.Table(user).ID(id).Update(data)
}

func (user *User) UpdatePassword(id int, password string) (int64, error)  {
    salt := user.generateSalt()
    safePassword := user.encryptPassword(password, salt)

    return user.Update(id, CommonMap{"password": safePassword, "salt": salt})
}

// 删除
func (user *User) Delete(id int) (int64, error) {
    return Db.Id(id).Delete(user)
}

// 禁用
func (user *User) Disable(id int) (int64, error) {
    return user.Update(id, CommonMap{"status": Disabled})
}

// 激活
func (user *User) Enable(id int) (int64, error) {
    return user.Update(id, CommonMap{"status": Enabled})
}

// 验证用户名和密码
func (user *User) Match(username, password string) bool {
    where := "(name = ? OR email = ?)"
    _, err := Db.Where(where, username, username).Get(user)
    if err != nil {
        return false
    }
    hashPassword := user.encryptPassword(password, user.Salt)
    if hashPassword != user.Password {
        return false
    }

    return true
}

// 用户名是否存在
func (user *User) UsernameExists(username string) (int64, error) {
    return Db.Where("name = ?", username).Count(user)
}

// 邮箱地址是否存在
func (user *User) EmailExists(email string) (int64, error) {
    return Db.Where("email = ?", email).Count(user)
}

func (user *User) List() ([]User, error) {
    list := make([]User, 0)
    err := Db.Desc("id").Find(&list)

    return list, err
}

func (user *User) Total() (int64, error) {
    return Db.Count(user)
}

// 密码加密
func (user *User) encryptPassword(password, salt string) string {
    return utils.Md5(password + salt)
}

// 生成密码盐值
func (user *User) generateSalt() string {
    return utils.RandString(PasswordSaltLength)
}
