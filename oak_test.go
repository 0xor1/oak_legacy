package oak

import(
	`bytes`
	`errors`
	`testing`
	`net/http`
	js `encoding/json`
	`net/http/httptest`
	`github.com/gorilla/mux`
	`github.com/gorilla/sessions`
	`github.com/stretchr/testify/assert`
)

func Test_create_without_existing_session(t *testing.T){
	w, r := setup(nil, nil, nil, _CREATE, ``)

	tr.ServeHTTP(w, r)

	resp := json{}
	readTestJson(w, &resp)
	assert.Equal(t, `test_entity_id`, resp[_ID].(string), `response json should contain the returned entityId`)
	assert.Equal(t, `test_creator_user_id`, tss.session.Values[_USER_ID], `session should have the provided user id`)
	assert.Equal(t, resp[_ID].(string), tss.session.Values[_ENTITY_ID].(string), `session should have a entityId matching the json response`)
	assert.Equal(t, tes.entity, tss.session.Values[_ENTITY].(*testEntity), `session should have the entity`)
}

func Test_create_with_existing_session(t *testing.T){
	w, r := setup(nil, nil, nil, _CREATE, ``)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	resp := json{}
	readTestJson(w, &resp)
	assert.Equal(t, `test_pre_set_entity_id`, resp[_ID].(string), `response json should have the existing entityId`)
	assert.Equal(t, `test_pre_set_user_id`, tss.session.Values[_USER_ID], `session should have the existing user id`)
	assert.Equal(t, resp[_ID].(string), tss.session.Values[_ENTITY_ID].(string), `session should have a entityId matching the json response`)
	assert.Equal(t, entity, tss.session.Values[_ENTITY], `session should have the existing entity`)
}

func Test_create_with_store_error(t *testing.T){
	w, r := setup(nil, nil, nil, _CREATE, ``)
	tes.createErr = errors.New(`test_create_error`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_create_error\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session.Values[_USER_ID], `session should not have a userId`)
	assert.Nil(t, tss.session.Values[_ENTITY_ID], `session should not have an entityId`)
	assert.Nil(t, tss.session.Values[_ENTITY], `session should not have an entity`)
}

func Test_join_without_existing_session(t *testing.T){
	w, r := setup(func(e Entity)map[string]interface{}{return json{"test": "yo"}}, nil, nil, _JOIN, `{"`+_ID+`":"req_test_entity_id"}`)
	tes.Create()

	tr.ServeHTTP(w, r)

	resp := json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getJoinResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
	assert.Equal(t, `test_user_id`, tss.session.Values[_USER_ID], `session should have the provided user id`)
	assert.Equal(t, `req_test_entity_id`, tss.session.Values[_ENTITY_ID].(string), `session should have the entityId`)
	assert.Equal(t, tes.entity, tss.session.Values[_ENTITY].(*testEntity), `session should have the entity`)
}

func Test_join_with_existing_session(t *testing.T){
	w, r := setup(func(e Entity)map[string]interface{}{return json{"test": "yo"}}, nil, nil, _JOIN, `{"`+_ID+`":"req_test_entity_id"}`)
	tes.Create()
	s, _ := tss.Get(r, ``)
	s.Values[_USER_ID] = `test_pre_set_user_id`
	s.Values[_ENTITY_ID] = `test_pre_set_entity_id`
	entity := &testEntity{}
	s.Values[_ENTITY] = entity

	tr.ServeHTTP(w, r)

	resp := json{}
	readTestJson(w, &resp)
	assert.Equal(t, `yo`, resp[`test`].(string), `response json should contain the returned data from getJoinResp`)
	assert.Equal(t, 0, int(resp[_VERSION].(float64)), `response json should contain the version number`)
	assert.Equal(t, `test_pre_set_user_id`, tss.session.Values[_USER_ID], `session should have the existing user id`)
	assert.Equal(t, `test_pre_set_entity_id`, tss.session.Values[_ENTITY_ID], `response json should have the existing entityId`)
	assert.Equal(t, entity, tss.session.Values[_ENTITY], `session should have the existing entity`)
}

func Test_join_with_request_missing_id(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{}`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, _ID + " value must be included in request\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_join_with_request_nonstring_id(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{"`+_ID+`": true}`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, _ID + " must be a string value\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_join_with_entity_store_read_error(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{"`+_ID+`": "test_entity_id"}`)
	tes.readErr = errors.New(`test_read_error`)

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_read_error\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

func Test_join_with_entity_store_update_error(t *testing.T) {
	w, r := setup(nil, nil, nil, _JOIN, `{"`+_ID+`": "test_entity_id"}`)
	tes.updateErr = errors.New(`test_update_error`)
	tes.entity = &testEntity{kick:func()bool{return true}}

	tr.ServeHTTP(w, r)

	assert.Equal(t, "test_update_error\n", w.Body.String(), `response body should be error message`)
	assert.Equal(t, 500, w.Code, `return code should be 500`)
	assert.Nil(t, tss.session, `session should not have been initialised`)
}

/**
 * helpers
 */

var tr *mux.Router

func readTestJson(w *httptest.ResponseRecorder, obj interface{}) error{
	return js.Unmarshal(w.Body.Bytes(), obj)
}

func setup(gjr GetJoinResp, gecr GetEntityChangeResp, pa PerformAct, path string, reqJson string) (*httptest.ResponseRecorder, *http.Request){
	tss = &testSessionStore{}
	tes = &testEntityStore{}
	tr = mux.NewRouter()
	Route(tr, tss, `test_session`, &testEntity{}, tes, gjr, gecr, pa)
	w := httptest.NewRecorder()
	var r *http.Request
	if reqJson != `` {
		r, _ = http.NewRequest(`POST`, path, bytes.NewBuffer([]byte(reqJson)))
	} else {
		r, _ = http.NewRequest(`POST`, path, nil)
	}
	return w, r
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

func (tes *testEntityStore) Create() (entityId string, entity Entity, err error) {
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

func (te *testEntity) RegisterNewUser() (string, error) {
	if te.registerNewUser != nil {
		return te.registerNewUser()
	}
	return `test_user_id`, nil
}

func (te *testEntity) UnregisterUser(userId string) error {
	if te.unregisterUser != nil {
		return te.unregisterUser(userId)
	}
	return nil
}

func (te *testEntity) Kick() bool {
	if te.kick != nil {
		return te.kick()
	}
	return false
}