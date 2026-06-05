package app

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// bcryptPrefix 用于识别 bcrypt 哈希，区分历史 MD5 数据。
const bcryptPrefix = "$2a$"

// hashPassword 新密码使用 bcrypt；bcrypt 失败时回退 MD5（极端情况）。
func hashPassword(raw string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		sum := md5.Sum([]byte(raw))
		return hex.EncodeToString(sum[:])
	}
	return string(hash)
}

// verifyPassword 兼容明文遗留、MD5 与 bcrypt 三种存储形态。
func verifyPassword(stored, provided string) bool {
	provided = strings.TrimSpace(provided)
	// 步骤 1：明文相等（历史数据）。
	if stored == provided {
		return true
	}
	// 步骤 2：bcrypt 校验。
	if strings.HasPrefix(stored, bcryptPrefix) {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(provided)) == nil
	}
	// 步骤 3：MD5 小写 hex 校验。
	return stored == legacyMD5(provided)
}

// legacyMD5 与 Java 旧系统一致的 MD5 hex（仅用于校验与懒升级源）。
func legacyMD5(raw string) string {
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// needsPasswordUpgrade 判断登录成功后是否应写回 bcrypt。
func needsPasswordUpgrade(stored string) bool {
	return !strings.HasPrefix(stored, bcryptPrefix)
}
