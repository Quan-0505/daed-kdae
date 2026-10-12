package graphql

import (
	"encoding/base64"
	"github.com/daeuniverse/dae-wing/db"
	"github.com/daeuniverse/dae-wing/pkg/sqlite"
	"golang.org/x/crypto/argon2"
	"gorm.io/gorm"
	"testing"
)

func legacyFixture(password string) string {
	salt := []byte("fixture-only-salt")
	hash := argon2.IDKey([]byte(password), salt, 2, 19456, 1, 32)
	return "$argon2id$v=19$m=19456,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash)
}
func TestLegacyPasswordRejectsWrongAndUnboundedParameters(t *testing.T) {
	encoded := legacyFixture("fixture-password")
	if !verifyLegacyPassword(encoded, "fixture-password") || verifyLegacyPassword(encoded, "wrong-password") {
		t.Fatal("password check failed")
	}
	for _, value := range []string{"invalid", "$argon2id$v=19$m=999999999,t=2,p=1$YWJjZGVmZ2hp$YWJjZA", "$argon2id$v=19$m=19456,t=2,p=0$YWJjZGVmZ2hp$YWJjZA"} {
		if verifyLegacyPassword(value, "fixture-password") {
			t.Fatal("unsafe parameters accepted")
		}
	}
}
func TestLegacyLoginUpgradesOnlyAfterSuccessfulVerification(t *testing.T) {
	d, err := gorm.Open(sqlite.Open(t.TempDir()+"/auth.db"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := d.DB()
	defer sql.Close()
	if err = d.AutoMigrate(&db.User{}); err != nil {
		t.Fatal(err)
	}
	old := legacyFixture("fixture-password")
	user := db.User{Username: "fixture-user", JwtSecret: "fixture-secret", PasswordHash: old}
	d.Create(&user)
	if _, err = getToken(d, user.Username, "wrong-password"); err == nil {
		t.Fatal("wrong password logged in")
	}
	var stored db.User
	d.First(&stored, user.ID)
	if stored.PasswordHash != old {
		t.Fatal("failed login changed hash")
	}
	token, err := getToken(d, user.Username, "fixture-password")
	if err != nil || token == "" {
		t.Fatal("legacy login failed")
	}
	d.First(&stored, user.ID)
	expected, _ := hashPassword([]byte(user.JwtSecret), "fixture-password")
	if stored.PasswordHash != expected || stored.JwtSecret != user.JwtSecret {
		t.Fatal("upgrade changed account identity or secret")
	}
	if _, err = getToken(d, user.Username, "fixture-password"); err != nil {
		t.Fatal("second Go login failed")
	}
}
