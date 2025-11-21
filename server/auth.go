package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/medatechnology/suresql"

	orm "github.com/medatechnology/simpleorm"

	"github.com/medatechnology/goutil/encryption"
	"github.com/medatechnology/goutil/medaerror"
	"github.com/medatechnology/goutil/medattlmap"
	"github.com/medatechnology/goutil/object"
)

// Constant for auth related like token settings
const (
	DECRYPT_FILLER          = "."
	TOKEN_STRING            = "token"
	TOKEN_LENGTH_MULTIPLIER = 3 // Controls token length/complexity

	// WRONG_PASSWORD_TEXT     = "password missmatch for user "
	// INVALID_CREDENTIAL_TEXT = "invalid credential"
)

// Global variables
var (
	// Instead of Redis, we use ttlmap is lighter
	// TokenMap        *medattlmap.TTLMap // For access tokens
	// RefreshTokenMap *medattlmap.TTLMap // For refresh tokens
	TokenStore TokenStoreStruct
)

// Mini Redis like Key-Value storage based on MedaTTLMap
type TokenStoreStruct struct {
	TokenMap        *medattlmap.TTLMap // For access tokens
	RefreshTokenMap *medattlmap.TTLMap // For refresh tokens
}

// InitTokenMaps initializes the token maps with default TTLs
func InitTokenMaps() {
	// Initialize with default expiration times
	TokenStore = NewTokenStore(suresql.DEFAULT_TOKEN_EXPIRES_MINUTES, suresql.DEFAULT_REFRESH_EXPIRES_MINUTES)
}

func NewTokenStore(exp, rexp time.Duration) TokenStoreStruct {
	return TokenStoreStruct{
		TokenMap:        medattlmap.NewTTLMap(exp, suresql.DEFAULT_TTL_TICKER_MINUTES),
		RefreshTokenMap: medattlmap.NewTTLMap(rexp, suresql.DEFAULT_TTL_TICKER_MINUTES),
	}
}

func (t TokenStoreStruct) GetAll() (map[string]interface{}, map[string]interface{}) {
	return t.TokenMap.Map(), t.RefreshTokenMap.Map()
}

// Check if tokenExist, if it is, return the value of the TokenMap[token] - which is interface{} type
func (t TokenStoreStruct) SaveToken(token suresql.TokenTable) {
	t.TokenMap.Put(token.Token, 0, token)
	t.RefreshTokenMap.Put(token.Refresh, 0, token)
}

// Check if tokenExist, if it is, return the value of the TokenMap[token] - which is interface{} type
func (t TokenStoreStruct) TokenExist(token string) (*suresql.TokenTable, bool) {
	val, ok := t.TokenMap.Get(token)
	// fmt.Println("All TokenMap:", t.TokenMap.Map())
	if !ok {
		return nil, false
	}
	// var tok TokenTable
	tok := val.(suresql.TokenTable)
	return &tok, true
}

// Check if tokenExist, if it is, return the value of the TokenMap[token] - which is interface{} type
func (t TokenStoreStruct) RefreshTokenExist(token string) (*suresql.TokenTable, bool) {
	val, ok := t.RefreshTokenMap.Get(token)
	if !ok {
		return nil, false
	}
	// var tok TokenTable
	tok := val.(suresql.TokenTable)
	return &tok, true
}

// This read from default _user table which is internal suresql table for username
// NOTE: Password is NOT cleared in this function - caller must clear it after use
func userNameExist(username string) (UserTable, error) {
	// Find user in database
	condition := orm.Condition{
		Field:    "username",
		Operator: "=",
		Value:    username,
	}

	var user UserTable
	userRecord, err := suresql.CurrentNode.InternalConnection.SelectOneWithCondition(user.TableName(), &condition)
	if err != nil {
		return user, err
	}

	// Convert to User struct
	user = object.MapToStructSlowDB[UserTable](userRecord.Data)
	// Password is intentionally kept for passwordMatch() validation
	// Callers MUST clear user.Password immediately after authentication
	return user, nil
}

func passwordMatch(user UserTable, pass string) error {
	encr, err := encryption.HashPin(pass, suresql.CurrentNode.Config.APIKey, suresql.CurrentNode.Config.ClientID)
	if err != nil {
		return err
	}
	if user.Password == encr {
		return nil
	} else {
		// return medaerror.NewMedaErrPtr(http.StatusUnauthorized, WRONG_PASSWORD_TEXT+user.Username, INVALID_CREDENTIAL_TEXT, nil)
		return errors.New("password mismatch for user " + user.Username)
	}
}

func createNewTokenResponse(user UserTable) suresql.TokenTable {
	var token suresql.TokenTable
	// Generate tokens using NewRandomTokenIterate with TOKEN_LENGTH_MULTIPLIER
	token.Token = encryption.NewRandomTokenIterate(TOKEN_LENGTH_MULTIPLIER)
	token.Refresh = encryption.NewRandomTokenIterate(TOKEN_LENGTH_MULTIPLIER)
	token.UserID = fmt.Sprintf("%d", user.ID)
	token.UserName = user.Username
	token.TokenExpiresAt = time.Now().Add(suresql.DEFAULT_TOKEN_EXPIRES_MINUTES)
	token.RefreshExpiresAt = time.Now().Add(suresql.DEFAULT_REFRESH_EXPIRES_MINUTES)

	// Store tokens in TTL maps with appropriate expiration times
	TokenStore.SaveToken(token)

	// Record token creation metric
	suresql.Metrics.RecordTokenCreated()

	// Return tokens in response
	return token
}

// ============= NOTE: this are not used at the moment, for future development where we encrypt the
// =============       data that is passed between request/response
//
// If we want to encrypt the credential for DB connect, encapsulated in just a string 'data'
// Maybe the encrypt from client is using PGP hence the Front-END will encrypt using public key
// And Backend decrypt with PrivateKey, Then the encrypted "data" is SureSQLConfig json
type EncryptedCredentials struct {
	Data string `json:"data"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func DecryptCredentials(data string, apiKey, clientID string) (*Credentials, error) {
	key := apiKey + DECRYPT_FILLER + clientID
	decrypted, err := encryption.DecryptWithKey(data, key)
	if err != nil {
		return nil, err
	}

	var creds Credentials
	err = json.Unmarshal([]byte(decrypted), &creds)
	if err != nil {
		return nil, err
	}

	return &creds, nil
}

// NOTE: maybe change this to just return empty string if error, then do checking if token=="" instead of error
func DecodeToken(tokenstring string, config *suresql.SureSQLDBMSConfig) (string, error) {
	tokenMap, err := encryption.ParseJWEToMap(tokenstring, []byte(config.JWEKey))
	if err != nil {
		return "", fmt.Errorf("failed to parse JWE token: %w", err)
	}

	// Check if token exists in map
	tokenValue, exists := tokenMap[TOKEN_STRING]
	if !exists {
		return "", medaerror.Simple("token not found in JWE payload")
	}

	if tokenValue != "HELLO_TEST" {
		return "", medaerror.Simple("token invalid: " + tokenValue)
	}

	config.Token = tokenValue
	return config.Token, nil
}
