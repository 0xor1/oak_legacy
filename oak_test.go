package oak

import(
	//`errors`
	`testing`
	`net/http`
	js `encoding/json`
	`net/http/httptest`
	`github.com/gorilla/mux`
	`github.com/gorilla/sessions`
	`github.com/stretchr/testify/assert`
)

func Test_play_with_no_existing_session(t *testing.T){
	tss = &testSessionStore{}
	tes = &testEntityStore{}
	router := mux.NewRouter()
	Route(router, tss, `test_session`, ) //TODO continue tomorrow
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(`POST`, _CREATE, nil)

	router.ServeHTTP(w, r)

	resp := json{}
	readTestJson(w, &resp)
	assert.Equal(t, `test_entity_id`, resp[_ID].(string), `response json should contain the returned entityId`)
	assert.Equal(t, `test_creator_user_id`, tss.session.Values[_USER_ID], `session should have the provided user id`)
	assert.Equal(t, resp[_ID].(string), tss.session.Values[_ID].(string), `session should have a entityId matching the json response`)
}

/**
 * helpers
 */

func readTestJson(w *httptest.ResponseRecorder, obj interface{}) error{
	return js.Unmarshal(w.Body.Bytes(), obj)
}

/**
 * Session
 */

var tss *testSessionStore

type testSessionStore struct{
	session *sessions.Session
}

func (tss *testSessionStore) Get(r *http.Request, sessName string) (*sessions.Session, error) {
	if tss.session == nil {
		tss.session = sessions.NewSession(tss, sessName)
	}
	return tss.session, nil
}

func (tss *testSessionStore) New(r *http.Request, sessName string) (*sessions.Session, error) {
	if tss.session == nil {
		tss.session = sessions.NewSession(tss, sessName)
	}
	return tss.session, nil
}

func (tss *testSessionStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	return nil
}

/**
 * Entity
 */

var tes *testEntityStore

type testEntityStore struct {
	entityId string
	entity *testEntity
	createErr error
	readErr error
	updateErr error
}

func (tes *testEntityStore) Create(r *http.Request) (entityId string, entity Entity, err error) {
	if tes.entity == nil {
		tes.entity = &testEntity{}
	}
	if tes.entityId == `` {
		tes.entityId = `test_entity_id`
	}
	return tes.entityId, tes.entity, tes.createErr
}

func (tes *testEntityStore) Read(entityId string) (Entity, error) {
	return tes.entity, tes.readErr
}

func (tes *testEntityStore) Update(entityId string, entity Entity) error {
	tes.entity = entity.(*testEntity)
	return tes.updateErr
}

type testEntity struct {
	getVersion func() int
	isActive func() bool
	createdBy func() string
	registerNewUser func() (string, error)
	unregisterUser func(string) error
	kick func() bool
}

func (te *testEntity) GetVersion() int {
	if te.getVersion != nil {
		return te.getVersion()
	}
	return 0
}

func (te *testEntity) IsActive() bool {
	if te.isActive != nil {
		return te.isActive()
	}
	return true
}

func (te *testEntity) CreatedBy() string {
	if te.createdBy != nil {
		return te.createdBy()
	}
	return `test_creator_user_id`
}

func (te *testEntity) RegisterNewUser() string {
	if te.registerNewUser != nil {
		return te.registerNewUser()
	}
	return `test_user_id`
}

func (te *testEntity) UnregisterUser() error {
	if te.unregisterUser != nil {
		return te.unregisterUser()
	}
	return nil
}

func (te *testEntity) Kick() {
	if te.kick != nil {
		return te.kick()
	}
	return false
}
