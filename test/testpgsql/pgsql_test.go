package testpgsql

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/vocdoni/go-dvote/crypto/ethereum"
	"gitlab.com/vocdoni/go-dvote/util"
	"gitlab.com/vocdoni/manager/manager-backend/config"
	"gitlab.com/vocdoni/manager/manager-backend/test/testcommon"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

var api testcommon.TestAPI

func TestMain(m *testing.M) {
	api = testcommon.TestAPI{Port: 12000 + rand.Intn(1000)}
	db := &config.DB{
		Dbname:   "vocdonimgr",
		Password: "vocdoni",
		Host:     "127.0.0.1",
		Port:     5432,
		Sslmode:  "disable",
		User:     "vocdoni",
	}
	if err := api.Start(db, ""); err != nil {
		log.Printf("SKIPPING: could not start the API: %v", err)
		return
	}
	os.Exit(m.Run())
	if err := api.DB.Ping(); err != nil {
		log.Printf("SKIPPING: could not connect to DB: %v", err)
		return
	}
	os.Exit(m.Run())
}

func TestEntity(t *testing.T) {
	entitySigner := ethereum.NewSignKeys()
	entitySigner.Generate()

	eid := entitySigner.Address().Bytes()
	entityID := ethereum.HashRaw(eid)
	entityIDStr := hex.EncodeToString(entityID)
	t.Log(entityIDStr)

	info := &types.EntityInfo{
		Address: eid,
		// Email:                   "entity@entity.org",
		Name:                    "test entity",
		CensusManagersAddresses: [][]byte{{1, 2, 3}},
		Origins:                 []types.Origin{types.Token},
	}

	err := api.DB.AddEntity(entityID, info)
	if err != nil {
		t.Fatalf("cannot add entity to the Postgres DB (pgsql.go:addEntity): %s", err)
	}

	entity, err := api.DB.Entity(entityID)
	if err != nil {
		t.Fatalf("cannot fetch entity from the Postgres DB (pgsql.go:Entity): %s", err)
	}
	marshalledEntityInfo, err := json.Marshal(entity.EntityInfo)
	if err != nil {
		t.Fatalf("cannot marshal retrieved Entity info: %s", err)
	}
	marshalledInfo, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("cannot marshal provided Entity info: %s", err)
	}
	if string(marshalledEntityInfo) != string(marshalledInfo) {
		t.Fatalf("expected %s info, but got %s", string(marshalledInfo), string(marshalledEntityInfo))
	}

	updateInfo := &types.EntityInfo{
		Address:                 eid,                         //same
		CensusManagersAddresses: [][]byte{{1, 2, 3}},         //same
		Origins:                 []types.Origin{types.Token}, //same
		Name:                    "test1 entity",
		CallbackURL:             "http://127.0.0.1/extapi",
		CallbackSecret:          "asdafgewgrf",
	}
	err = api.DB.UpdateEntity(entityID, updateInfo)
	if err != nil {
		t.Fatalf("cannot update entity Entity info: (%s)", err)
	}
	updatedEntity, err := api.DB.Entity(entityID)
	if err != nil {
		t.Fatalf("cannot fetch entity from the Postgres DB (pgsql.go:Entity): %s", err)
	}
	marshalledUpdatedEntityInfo, err := json.Marshal(updatedEntity.EntityInfo)
	if err != nil {
		t.Fatalf("cannot marshal retrieved Entity info: %s", err)
	}
	marshalleUpdatedInfo, err := json.Marshal(updateInfo)
	if err != nil {
		t.Fatalf("cannot marshal provided Entity info: %s", err)
	}
	if string(marshalleUpdatedInfo) != string(marshalledUpdatedEntityInfo) {
		t.Fatalf("expected \n%s info, but got \n%s", string(marshalleUpdatedInfo), string(marshalledUpdatedEntityInfo))
	}

	if updatedEntity.IsAuthorized {
		t.Fatalf("error entity authorized not explicitly: (%v)", err)
	}

	if err = api.DB.AuthorizeEntity(entityID); err != nil {
		t.Fatalf("error authorizing entity: (%v)", err)
	}
	updatedEntity, err = api.DB.Entity(entityID)
	if err != nil {
		t.Fatalf("cannot fetch entity from the Postgres DB (pgsql.go:Entity): %s", err)
	}
	if !updatedEntity.IsAuthorized {
		t.Fatalf("error entity not authorized even though authorizedEntity was succesful: (%v)", err)
	}

	err = api.DB.AuthorizeEntity(entityID)
	if err == nil {
		t.Fatalf("managed to reauthorize entity: (%v)", err)
	} else if err.Error() != "already authorized" {
		t.Fatalf("error while trying to reauthorize entity: (%v)", err)
	}

	entities, err := api.DB.EntitiesID()
	t.Logf("entities: %+v", entities)
	if err != nil {
		t.Fatalf("cannot get entities from the Postgres DB (pgsql.go:Entities): %s", err)
	}
	if len(entities) != 1 {
		t.Fatalf("expected one entity, got %d", len(entities))
	}
}

func TestUser(t *testing.T) {
	var err error
	userSigner := ethereum.NewSignKeys()
	userSigner.Generate()
	user := &types.User{PubKey: userSigner.Public.X.Bytes()}
	err = api.DB.AddUser(user)
	if err != nil {
		t.Fatalf("cannot add user to the Postgres DB (pgsql.go:addUser): %s", err)
	}
	_, err = api.DB.User(userSigner.Public.X.Bytes())
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:User): %s", err)
	}
}

func TestMember(t *testing.T) {
	var id uuid.UUID
	var initialCount, count int
	db := api.DB
	// Create or retrieve existing entity
	// create entity
	_, entities := testcommon.CreateEntities(2)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// Create pubkey and Add membmer to the db
	memberSigner := ethereum.NewSignKeys()
	memberSigner.Generate()
	pub, _ := memberSigner.HexString()
	pub, _ = ethereum.DecompressPubKey(pub)
	pubBytes, err := hex.DecodeString(pub)
	if err != nil {
		log.Fatalf("%s", err)
	}
	memberInfo := &types.MemberInfo{}
	memberInfo.FirstName = "Lak"
	memberInfo.LastName = "Lik"
	memberInfo.DateOfBirth.Round(time.Microsecond).UTC()
	memberInfo.Verified.Round(time.Microsecond)
	user := &types.User{PubKey: pubBytes}
	err = api.DB.AddUser(user)
	if err != nil {
		t.Fatalf("cannot add user to the Postgres DB (pgsql.go:addUser) %s", err)
	}
	if initialCount, err = api.DB.CountMembers(entities[0].ID); err != nil {
		t.Fatalf("cannot count members correctly: %+v", err)
	}

	id, err = api.DB.AddMember(entities[0].ID, pubBytes, memberInfo)
	if err != nil {
		t.Fatalf("cannot add member to the Postgres DB (pgsql.go:addMember): %s", err)
	}

	if count, err = api.DB.CountMembers(entities[0].ID); err != nil || count != initialCount+1 {
		t.Fatalf("expected %d counted: %d\ncannot count members correctly: %+v", initialCount+1, count, err)
	}

	// cannot add twice
	id2, err := api.DB.AddMember(entities[0].ID, pubBytes, memberInfo)
	if id2 != uuid.Nil {
		t.Fatalf("cannot add member twice to the Postgres DB (pgsql.go:addMember): %s", err)
	}

	// Query by ID
	member, err := db.Member(entities[0].ID, &id)
	if err != nil {
		t.Errorf("Error retrieving member from the Postgres DB (pgsql.go:Member): %s", err)
	}

	// Query by Public Key
	memberPubKey, err := db.MemberPubKey(entities[0].ID, pubBytes)
	if err != nil {
		t.Fatalf("cannot fetch member from the Postgres DB (pgsql.go:MemberPubKey): %s", err)
	}
	if memberPubKey.ID != member.ID {
		t.Fatalf("error retrieving member using MemberPubKey")
	}
	// Check first timestamps that need different handling
	// and then assing one to another so that the rest of test doesnt fail
	if !memberInfo.Verified.Equal(member.Verified) {
		t.Fatalf("expected %s verified on member info, but got %s", memberInfo.Verified, member.Verified)
	}
	if !memberInfo.DateOfBirth.Equal(member.DateOfBirth) {
		t.Fatalf("expected %s dateofbirth on member info, but got %s", memberInfo.DateOfBirth, member.DateOfBirth)
	}
	memberInfo.DateOfBirth = member.DateOfBirth
	memberInfo.Verified = member.Verified

	// Check fetchd data match send data
	marshalledMemberInfo, err := json.Marshal(member.MemberInfo)
	if err != nil {
		t.Fatalf("cannot marshal member info from query: %s", err)
	}
	marshalledInfo, err := json.Marshal(memberInfo)
	if err != nil {
		t.Fatalf("cannot marshal member info: %s", err)
	}

	if string(marshalledMemberInfo) != string(marshalledInfo) {
		t.Fatalf("expected %s marshaled member info, but got %s", marshalledMemberInfo, marshalledInfo)
	}

	// Query by UUID
	member, err = db.Member(entities[0].ID, &member.ID)
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:Member): %s", err)
	}
	if !bytes.Equal(member.PubKey, pubBytes) {
		t.Fatalf("expected %s member pubkey, but got %s", member.PubKey, pubBytes)
	}

	// Test SetMemberInfo
	newInfo := &types.MemberInfo{Email: "updated@mail.com", FirstName: ""}
	err = api.DB.UpdateMember(entities[0].ID, &member.ID, newInfo)
	if err != nil {
		t.Fatalf("cannot update user info to the Postgres DB (pgsql.go:updateMember): %s", err)
	}
	newMember, err := db.Member(entities[0].ID, &member.ID)
	if err != nil {
		t.Fatalf("cannot fetch user from the Postgres DB (pgsql.go:Member): %s", err)
	}
	if newMember.Email != "updated@mail.com" {
		t.Fatal("updateMember failed to update member Email in the Postgres DB (pgsql.go:Member)")
	}
	if newMember.FirstName != "Lak" {
		t.Fatal("updateMember with an empty string ovewrites the Name field while it shouldn't(pgsql.go:updateMember)")
	}

	// Test Bulk Info
	var bulkMembers []types.MemberInfo
	for i := 0; i < 10; i++ {
		info := types.MemberInfo{FirstName: fmt.Sprintf("Name%d", i), LastName: fmt.Sprintf("LastName%d", i)}
		bulkMembers = append(bulkMembers, info)
	}
	err = api.DB.ImportMembers(entities[0].ID, bulkMembers)
	if err != nil {
		t.Fatalf("cannot add members to Postgres DB (pgsql.go:AddMemberBulk): %s", err)
	}

	if count, err = api.DB.CountMembers(entities[0].ID); err != nil || count != initialCount+11 {
		t.Fatalf("expected %d counted: %d\ncannot count members correctly: %+v", initialCount+11, count, err)
	}

	// Test Selecting all members
	allMembers, err := api.DB.ListMembers(entities[0].ID, &types.ListOptions{})
	if err != nil {
		t.Fatalf("cannot select all members from Postgres DB (pgsql.go:ListMembers): %s", err)
	}

	// Test Selecting filtered members
	limit := 5
	filter := &types.ListOptions{
		Skip:   2,
		Count:  limit,
		SortBy: "lastName",
		Order:  "descend",
	}
	members, err := api.DB.ListMembers(entities[0].ID, filter)
	if err != nil {
		t.Fatalf("cannot select all members from Postgres DB (pgsql.go:ListMembers): %s", err)
	}
	if len(members) > limit {
		t.Fatalf("expected limit to be less than members length, %d <= %d", len(members), limit)
	}

	// Test Selecting all members and retrieving just their uuids and emails
	tokenMembers, err := api.DB.MembersTokensEmails(entities[0].ID)
	if err != nil {
		t.Fatal("cannot fetch tokens and emails from the Prostgres DB (pgsql.go:MembersTokensEmails)")
	}
	if len(tokenMembers) != len(bulkMembers) {
		t.Fatalf("Expected retrieving tokens for %d members but instead retrieved %d (pgsql.go:MembersTokensEmails)", len(allMembers), len(tokenMembers))
	}

	// Test deleting member
	// 1. Can delete existing member
	if err := api.DB.DeleteMember(entities[0].ID, &member.ID); err != nil {
		t.Fatalf("error deleting member %+v", err)
	}
	if _, err = db.Member(entities[0].ID, &member.ID); err != sql.ErrNoRows {
		t.Fatalf("error retrieving deleting member %+v", err)
	}

	// 2. Get error deleting inexisting member
	if err := api.DB.DeleteMember(entities[0].ID, &uuid.UUID{}); err == nil {
		t.Fatalf("managed to delete random member %+v", err)
	}

	// Test Register Flow
	if err := api.DB.AddEntity(entities[1].ID, &entities[1].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}
	_, registerMembers, err := testcommon.CreateMembers(entities[1].ID, 1)
	if err != nil {
		t.Fatalf("cannot create members: %s", err)
	}
	n := 5
	tokens, err := api.DB.CreateNMembers(entities[1].ID, n)
	if err != nil {
		t.Fatalf("unable to create members using CreateNMembers:  (%+v)", err)
	}
	// 0. Checking that the user did not exist already
	_, err = api.DB.User(registerMembers[0].PubKey)
	if err != sql.ErrNoRows {
		t.Fatalf("unable to retrieve registered member:  (%+v)", err)
	}
	// 1. Registering member
	if err := api.DB.RegisterMember(entities[1].ID, registerMembers[0].PubKey, &tokens[0]); err != nil {
		t.Fatalf("unable to register member using existing token:  (%+v)", err)
	}
	// 2. Checking that the member was created with right values
	registeredMember, err := api.DB.Member(entities[1].ID, &tokens[0])
	if err != nil {
		t.Fatalf("unable to retrieve registered member:  (%+v)", err)
	}
	if hex.EncodeToString(registeredMember.PubKey) != hex.EncodeToString(registerMembers[0].PubKey) {
		t.Fatalf("unable to set registered members pubkey. Expected: %q got  %s", hex.EncodeToString(registerMembers[0].PubKey), hex.EncodeToString(registeredMember.PubKey))
	}
	// 3. Checking that the corresponding User was also created
	registeredUser, err := api.DB.User(registerMembers[0].PubKey)
	if err != nil {
		t.Fatalf("unable to retrieve registered user:  (%+v)", err)
	}
	if hex.EncodeToString(registeredUser.PubKey) != hex.EncodeToString(registerMembers[0].PubKey) {
		t.Fatalf("unable to set registered users pubke %s", hex.EncodeToString(member.PubKey))
	}
	entityID := hex.EncodeToString(entities[1].ID)
	t.Logf("entityID %s", entityID)
	randToken := tokens[1].String()
	t.Logf("token: %s", randToken)
	// 4. Not allowing register of random token
	u := uuid.New()
	if err := api.DB.RegisterMember(entities[1].ID, registerMembers[0].PubKey, &u); err == nil {
		t.Fatalf("able to register member using random token:  (%+v)", err)
	}
	// 5. Not allowing using existing token with different entity ID
	if err := api.DB.RegisterMember(entities[0].ID, registerMembers[0].PubKey, &tokens[1]); err == nil {
		t.Fatalf("able to register member using existing token but to non-correspondig entity:  (%+v)", err)
	}

}

func TestTarget(t *testing.T) {
	var inTarget, outTarget *types.Target
	var targets []types.Target
	var targetID uuid.UUID
	var err error
	var count int

	// create entity
	_, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// Check able to list 0 targets
	if targets, err := api.DB.ListTargets(entities[0].ID); err != nil || len(targets) != 0 {
		t.Fatalf("errors retrieving all targets: %s", err)
	}

	//Verify that 0 targets are counted
	if count, err = api.DB.CountTargets(entities[0].ID); err != nil || count != 0 {
		t.Fatalf("expected %d counted: %d\ncannot count targets correctly: %+v", 0, count, err)
	}

	inTarget = &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	// test adding target
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err != nil {
		t.Fatalf("cannot add created target into database: %s", err)
	}

	// retrieve added target
	if outTarget, err = api.DB.Target(entities[0].ID, &targetID); err != nil || outTarget.Name != inTarget.Name {
		t.Fatalf("error retrieving created target from database: %s", err)
	}

	// Check not allowing duplicate targets
	if targetID, err = api.DB.AddTarget(entities[0].ID, inTarget); err == nil {
		t.Fatalf("able to add duplicate target into database: %s", err)
	}

	// Check able to list all (1 for now) targets
	if targets, err = api.DB.ListTargets(entities[0].ID); err != nil || len(targets) != 1 {
		t.Fatalf("errors retrieving all targets: %s", err)
	}

	//Verify that 0 targets are counted
	if count, err := api.DB.CountTargets(entities[0].ID); err != nil || count != 1 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count targets correctly: %+v", err)
	}
}

func TestCensus(t *testing.T) {
	var root, idBytes []byte
	var err error
	// create entity
	_, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// add target
	target := &types.Target{EntityID: entities[0].ID, Name: "all", Filters: json.RawMessage([]byte("{}"))}

	var targetID uuid.UUID
	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add target into database: %s", err)
	}

	// Genrate ID and root
	id := util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	if root, err = hex.DecodeString(util.RandomHex(len(entities[0].ID))); err != nil {
		t.Fatalf("cannot generate root: %s", err)
	}
	name := fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))
	// create census info
	censusInfo := &types.CensusInfo{
		Name:          name,
		MerkleRoot:    root,
		MerkleTreeURI: fmt.Sprintf("ipfs://%s", util.TrimHex(id)),
	}

	//Verify that 0 census are counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 0 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	if err := api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo); err != nil {
		t.Fatalf("cannot add census into database: %s", err)
	}

	//Verify that census exists
	if census, err := api.DB.Census(entities[0].ID, idBytes); err != nil || census.Name != name {
		t.Fatalf("unable to recover created census: %s", err)
	}

	//Verify that one census is counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 1 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	//Verify that cannot add duplicate census
	if err := api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo); err == nil {
		t.Fatal("able to create duplicate census into database")
	}

	//Verify that cannot add  census for inexisting target
	id = util.RandomHex(len(entities[0].ID))
	if idBytes, err = hex.DecodeString(util.TrimHex(id)); err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	if err := api.DB.AddCensus(entities[0].ID, idBytes, &uuid.UUID{}, censusInfo); err == nil {
		t.Fatal("able to create census for inexisting target (pgsql.go:AddCensus)")
	}

	//Add second census (needs second target)
	target = &types.Target{EntityID: entities[0].ID, Name: "all1", Filters: json.RawMessage([]byte("{}"))}

	targetID, err = api.DB.AddTarget(entities[0].ID, target)
	if err != nil {
		t.Fatalf("cannot add second target into database: %s", err)
	}
	id = util.RandomHex(len(entities[0].ID))
	idBytes, err = hex.DecodeString(util.TrimHex(id))
	if err != nil {
		t.Fatalf("cannot decode random id: %s", err)
	}
	censusInfo.Name = fmt.Sprintf("census%s", strconv.Itoa(rand.Int()))

	err = api.DB.AddCensus(entities[0].ID, idBytes, &targetID, censusInfo)
	if err != nil {
		t.Fatal("unable to create second census (pgsql.go:AddCensus)")
	}

	//Verify that one census is counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 2 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	var censuses []types.Census
	censuses, err = api.DB.ListCensus(entities[0].ID, &types.ListOptions{})
	if err != nil || len(censuses) != 2 {
		t.Fatal("unable to list censuses correctly (pgsql.go:Censuses)")
	}

	// Verify that an existing census can be deleted
	if err = api.DB.DeleteCensus(entities[0].ID, censuses[0].ID); err != nil {
		t.Fatalf("cannot delete census correctly (pgsql.go:DeleteCensus): %v", err)
	}

	census, err := api.DB.Census(entities[0].ID, censuses[0].ID)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("error in checking wether census was deleted correctly (pgsql.go:DeleteCensus): %v", err)
	}
	if err == nil || census != nil {
		t.Fatal(" census was not deleted correctly (pgsql.go:DeleteCensus)")
	}

	//Verify that one census is counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 1 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	// Verify that a random non-existing census cannot be deleted
	randomCensusID := util.RandomHex(len(censuses[0].ID))
	randomCensusIDBytes, _ := hex.DecodeString(randomCensusID)
	if err = api.DB.DeleteCensus(entities[0].ID, randomCensusIDBytes); err == nil {
		t.Fatalf("managed to delete a census with a random generated census ID (pgsql.go:DeleteCensus): %v", err)
	}

	//Verify that one census is counted
	if count, err := api.DB.CountCensus(entities[0].ID); err != nil || count != 1 {
		t.Errorf("counted %d", count)
		t.Fatalf("cannot count censuses correctly: %+v", err)
	}

	// Verify that the corresponding members are also deleted

}

func TestTags(t *testing.T) {
	// create entity
	_, entities := testcommon.CreateEntities(1)
	// add entity
	if err := api.DB.AddEntity(entities[0].ID, &entities[0].EntityInfo); err != nil {
		t.Fatalf("cannot add created entity into database: %s", err)
	}

	// create members
	// Test that members without keys are not counted
	memberIDs, err := api.DB.CreateNMembers(entities[0].ID, 2)
	if err != nil {
		t.Fatalf("cannot generate random members (%v)", err)
	}

	// Add tag
	tag_id, err := api.DB.AddTag(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("error creating tag:  (%v)", err)
	}
	tag, err := api.DB.Tag(entities[0].ID, tag_id)
	if err != nil {
		t.Fatalf("error retrieving newly created tag:  (%v)", err)
	}
	// Add tag to members
	added, err := api.DB.AddTagToMembers(entities[0].ID, memberIDs, tag.ID)
	if err != nil || added != int64(len(memberIDs)) {
		t.Fatalf("unable to add member tags:  (%v)", err)
	}
	// verify tags were registered correctly
	taggedMembers, err := api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("error retrieving entity members:  (%v)", err)
	}
	for _, member := range taggedMembers {
		// since it is the only tag of the elements it is enough to check array size
		if len(member.Tags) != 1 || member.Tags[0] != tag.ID {
			t.Fatalf("Did not update correctly member tags")
		}
	}

	// verify that the same tag cannot be added twice
	// Add tag to members
	updated, err := api.DB.AddTagToMembers(entities[0].ID, memberIDs, tag.ID)
	if err == nil || updated != 0 {
		t.Fatalf("able to add the same member tag twice:  (%v)", err)
	}

	// delete added tag
	deleted, err := api.DB.RemoveTagFromMembers(entities[0].ID, memberIDs, tag.ID)
	if err != nil || deleted != int64(len(memberIDs)) {
		t.Fatalf("unable to delete member tags:  (%v)", err)
	}
	// verify tags were removed correctly
	taggedMembers, err = api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("error retrieving entity members:  (%v)", err)
	}
	for _, member := range taggedMembers {
		// since it is the only tag of the elements it is enough to check array size
		if len(member.Tags) != 0 {
			t.Fatalf("Did not delete correctly member tags")
		}
	}

	// add again tag and delete the tag (removing it automatically from the members that have it)
	added, err = api.DB.AddTagToMembers(entities[0].ID, memberIDs, tag.ID)
	if err != nil || added != int64(len(memberIDs)) {
		t.Fatalf("unable to add member tags:  (%v)", err)
	}
	if err = api.DB.DeleteTag(entities[0].ID, tag.ID); err != nil {
		t.Fatalf("unable to delete tag that exists for members:  (%v)", err)
	}
	taggedMembers, err = api.DB.ListMembers(entities[0].ID, nil)
	if err != nil {
		t.Fatalf("error retrieving entity members:  (%v)", err)
	}
	for _, member := range taggedMembers {
		// since it is the only tag of the elements it is enough to check array size
		if len(member.Tags) != 0 {
			t.Fatalf("Did not cascade delete correctly member tags")
		}
	}
	if _, err := api.DB.Tag(entities[0].ID, tag.ID); err == nil || err != sql.ErrNoRows {
		t.Fatalf("tag was not deleted correctly %v", err)
	}

	// add again tag and delete the tag
	tag_id, err = api.DB.AddTag(entities[0].ID, "TestTag")
	if err != nil {
		t.Fatalf("error creating tag:  (%v)", err)
	}
	if err = api.DB.DeleteTag(entities[0].ID, tag_id); err != nil {
		t.Fatalf("unable to delete tag that exists for members:  (%v)", err)
	}
	if _, err := api.DB.Tag(entities[0].ID, tag.ID); err == nil || err != sql.ErrNoRows {
		t.Fatalf("tag was not deleted correctly %v", err)
	}
}
