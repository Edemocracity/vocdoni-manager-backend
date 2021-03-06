package testcommon

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"time"

	randomdata "github.com/Pallinder/go-randomdata"

	"go.vocdoni.io/dvote/crypto/ethereum"
	util "go.vocdoni.io/dvote/util"
	"go.vocdoni.io/manager/types"
)

// CreateEntities a given number of random entities
func CreateEntities(size int) ([]*ethereum.SignKeys, []*types.Entity) {
	var entityID, entityAddress []byte
	var err error
	signers := CreateEthRandomKeysBatch(size)
	mp := make([]*types.Entity, size)
	for i := 0; i < size; i++ {
		// retrieve entity ID
		entityAddress, err = hex.DecodeString(util.TrimHex(signers[i].Address().String()))
		if err != nil {
			return nil, nil
		}
		entityID = ethereum.HashRaw(entityAddress)
		mp[i] = &types.Entity{
			ID: entityID,
			EntityInfo: types.EntityInfo{
				Address:                 entityAddress,
				Email:                   randomdata.Email(),
				Name:                    randomdata.FirstName(2),
				CensusManagersAddresses: [][]byte{{1, 2, 3}},
				Origins:                 []types.Origin{types.Token},
				CallbackURL:             "",
				CallbackSecret:          "",
			},
		}
	}
	return signers, mp
}

// CreateMembers a given number of members with its entityID set to entityID
func CreateMembers(entityID []byte, size int) ([]*ethereum.SignKeys, []*types.Member, error) {
	signers := CreateEthRandomKeysBatch(size)
	members := make([]*types.Member, size)
	// if membersInfo not set generate random data
	for i := 0; i < size; i++ {
		pub, _ := signers[i].HexString()
		pub, _ = ethereum.DecompressPubKey(pub)
		pubBytes, err := hex.DecodeString(pub)
		if err != nil {
			return nil, nil, err
		}
		members[i] = &types.Member{
			EntityID: entityID,
			PubKey:   pubBytes,
			MemberInfo: types.MemberInfo{
				DateOfBirth:   RandDate(),
				Email:         randomdata.Email(),
				FirstName:     randomdata.FirstName(2),
				LastName:      randomdata.LastName(),
				Phone:         randomdata.PhoneNumber(),
				StreetAddress: randomdata.Address(),
				Consented:     RandBool(),
				// Verified:      RandDate(),
				Origin:       types.Token,
				CustomFields: json.RawMessage([]byte("{}")),
			},
		}
	}
	return signers, members, nil
}

// CreateEthRandomKeysBatch creates a set of eth random signing keys
func CreateEthRandomKeysBatch(n int) []*ethereum.SignKeys {
	s := make([]*ethereum.SignKeys, n)
	for i := 0; i < n; i++ {
		s[i] = ethereum.NewSignKeys()
		if err := s[i].Generate(); err != nil {
			return nil
		}
	}
	return s
}

// RandDate creates a random date
func RandDate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min
	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}

// RandBool creates a random bool
func RandBool() bool {
	return rand.Float32() < 0.5
}
