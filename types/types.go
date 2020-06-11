package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CreatedUpdated struct {
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type Entity struct {
	CreatedUpdated
	ID []byte `json:"id" db:"id"`
	EntityInfo
}

type EntityInfo struct {
	Address                 []byte   `json:"address" db:"address"`
	Email                   string   `json:"email,omitempty" db:"email"`
	Name                    string   `json:"name" db:"name"`
	CensusManagersAddresses [][]byte `json:"censusManagersAddresses,omitempty" db:"census_managers_addresses"`
	Origins                 []Origin `json:"origin" db:"origin"`
}

type origin int

const (
	Token origin = iota
	Form
	DB
)

type Origin interface {
	Origin() origin
}

// every base must fulfill the Baser interface
func (b origin) Origin() origin {
	return b
}

func (b origin) String() string {
	return [...]string{"Token", "Form", "DB"}[b]
}

func ToOrigin(origin string) Origin {
	switch origin {
	case "Token":
		return Token.Origin()
	case "Form":
		return Form.Origin()
	case "DB":
		return DB.Origin()
	default:
		return nil
	}
}

type Member struct {
	CreatedUpdated
	ID       uuid.UUID `json:"id" db:"id"`
	EntityID []byte    `json:"entityId" db:"entity_id"`
	PubKey   []byte    `json:"publicKey" db:"public_key"`
	MemberInfo
}

type MemberInfo struct {
	DateOfBirth   time.Time       `json:"dateOfBirth,omitempty" db:"date_of_birth"`
	Email         string          `json:"email,omitempty" db:"email"`
	FirstName     string          `json:"firstName,omitempty" db:"first_name"`
	LastName      string          `json:"lastName,omitempty" db:"last_name"`
	Phone         string          `json:"phone,omitempty" db:"phone"`
	StreetAddress string          `json:"streetAddress,omitempty" db:"street_address"`
	Consented     bool            `json:"consented" db:"consented"`
	Verified      time.Time       `json:"verified,omitempty" db:"verified"`
	Origin        Origin          `json:"origin,omitempty" db:"origin"`
	CustomFields  json.RawMessage `json:"customFields" db:"custom_fields"`
}

// In case COPY FROM is adopted
// func (m *MemberInfo) GetDBFields() []string {
// 	return []string{
// 		"date_of_birth",
// 		"email",
// 		"first_name",
// 		"last_name",
// 		"phone",
// 		"street_address",
// 		"consented",
// 		"verified",
// 		"origin",
// 		"custom_fields",
// 	}
// }

// func (m *MemberInfo) GetActiveDBFields() map[string]interface{} {
// 	ret := make(map[string]interface{})
// 	str := reflect.Indirect(reflect.ValueOf(m))
// 	// var fields []string
// 	for i := 0; i < str.NumField(); i++ {
// 		ret[str.Type().Field(i).Tag.Get("db")] = str.Field(i).Interface()
// 		// fields = append(fields, str.Field(i).Name)
// 	}
// 	return ret
// }

// func (m *MemberInfo) GetRecord() []interface{} {
// 	var list []interface{}
// 	generic := reflect.Indirect(reflect.ValueOf(MemberInfo{}))
// 	totalFields := m.GetDBFields()
// 	activeFields := m.GetActiveDBFields()
// 	// record := reflect.Indirect(reflect.ValueOf(m))
// 	for _, field := range totalFields {
// 		// TODO check ommited
// 		data, ok := activeFields[field]
// 		if ok {
// 			list = append(list, data)
// 		} else {
// 			// generic.FieldByName(field).Type().
// 			list = append(list, generic.FieldByName(field).Interface())
// 		}

// 		// list = append(list, record.FieldByName(field).Elem)
// 	}
// 	return list
// }

// func (m *MemberInfo) Normalize() {
// 	if m.CustomFields == nil {
// 		m.CustomFields = []byte{}
// 	}
// }

type User struct {
	PubKey         []byte `json:"publicKey" db:"public_key"`
	DigestedPubKey []byte `json:"digestedPublicKey" db:"digested_public_key"`
}

type Census struct {
	EntityID []byte `json:"entityId" db:"entityId"`
	ID       []byte `json:"id" db:"id"`
	TargetID string `json:"targetId" db:"targetId"`
	CensusInfo
}

type CensusInfo struct {
	Created time.Time `json:"created,omitempty"`
	Name    string    `json:"name,omitempty"`
	Root    []byte    `json:"root,omitempty"`
	URI     string    `json:"uri,omitempty"`
}

type Target struct {
	ID       string            `json:"id" db:"id"`
	EntityID []byte            `json:"entityId" db:"entityId"`
	Name     string            `json:"name" db:"name"`
	Filters  map[string]string `json:"filters" db:"filters"`
}
