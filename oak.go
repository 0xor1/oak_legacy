package oak

import(
	`errors`
	`net/http`
	`encoding/gob`
	js `encoding/json`
	`github.com/0xor1/sus`
	`github.com/gorilla/mux`
	`github.com/gorilla/sessions`
)

const (
	_CREATE = `/create`
	_JOIN 	= `/join`
	_POLL 	= `/poll`
	_ACT 	= `/act`

	_USER_ID	= `userId`
	_ENTITY_ID	= `entityId`
	_ENTITY		= `entity`

	_ID			= `id`
	_VERSION	= `v`
	_NO_CHANGE	= `noChange`
)

type EntityStore interface{
	Create() (userId string, entityId string, entity Entity, err error)
	Read(entityId string) (entity Entity, err error)
	Update(entityId string, entity Entity) (err error)
}

type Entity interface {
	sus.Version
	IsActive() bool
	RegisterNewUser() (userId string, err error)
	Kick() (updated bool)
	Act(action interface{}) error
}

type GetJoinResp func(e Entity) map[string]interface{}
type GetEntityChangeResp func(e Entity) map[string]interface{}
type GetActParam func(r *http.Request) (param interface{}, err error)

var (
	sessionStore		sessions.Store
	sessionName			string
	entityStore			EntityStore
	getJoinResp			GetJoinResp
	getEntityChangeResp	GetEntityChangeResp
	getActParam			GetActParam
)

func Route(router *mux.Router, sessStore sessions.Store, sessName string, e Entity, es EntityStore, gjr GetJoinResp, gecr GetEntityChangeResp, gap GetActParam){
	gob.Register(e)
	sessionStore = sessStore
	sessionName = sessName
	entityStore = es
	getJoinResp = getJoinResp
	getEntityChangeResp = gecr
	getActParam = gap
	router.Path(_CREATE).HandlerFunc(create)
	router.Path(_JOIN).HandlerFunc(join)
	router.Path(_POLL).HandlerFunc(poll)
	router.Path(_ACT).HandlerFunc(act)
}

func create(w http.ResponseWriter, r *http.Request){
	s, _ := getSession(w, r)
	if s.isNotEngaged() {
		userId, entityId, entity, err := entityStore.Create()
		if err != nil {
			writeError(w, err)
			return
		}
		s.set(userId, entityId, entity)
	}
	writeJson(w, &json{_ID: s.getEntityId()})
}

func join(w http.ResponseWriter, r *http.Request) {
	entityId, _, err := getRequestData(r, false)
	if err != nil {
		writeError(w, err)
		return
	}

	entity, err := fetchEntity(entityId)
	if err != nil {
		writeError(w, err)
		return
	}

	s, _ := getSession(w, r)
	if s.isNotEngaged() && entity.IsActive() {
		if userId, err := entity.RegisterNewUser(); err != nil {
			writeError(w, err)
			return
		} else if err := entityStore.Update(entityId, entity); err != nil {
			writeError(w, err)
			return
		} else {
			//entity was updated successfully this user is now active in this entity
			s.set(userId, entityId, entity)
		}
	}

	respJson := getJoinResp(entity)
	respJson[_VERSION] = entity.GetVersion()
	writeJson(w, &respJson)
}

func poll(w http.ResponseWriter, r *http.Request) {
	entityId, version, err := getRequestData(r, true)
	if err != nil {
		writeError(w, err)
		return
	}

	entity, err := fetchEntity(entityId)
	if err != nil {
		writeError(w, err)
		return
	}

	if version == entity.GetVersion() {
		writeJson(w, &json{_NO_CHANGE: true})
		return
	} else {
		s, _ := getSession(w, r)
		if s.getEntityId() == entityId {
			if entity.IsActive() {
				s.set(s.getUserId(), entityId, entity)
			} else {
				s.clear()
			}
		}
		respJson := getEntityChangeResp(entity)
		respJson[_VERSION] = entity.GetVersion()
		writeJson(w, &respJson)
	}
}

func act(w http.ResponseWriter, r *http.Request) {
	s, _ := getSession(w, r)
	sessionEntity := s.getEntity()
	if sessionEntity == nil {
		writeError(w, errors.New(`no entity in session`))
		return
	}

	actParam, err := getActParam(r)
	if err != nil {
		writeError(w, err)
		return
	}

	if err = sessionEntity.Act(actParam); err != nil {
		writeError(w, err)
		return
	}

	entityId := s.getEntityId()
	entity, err := fetchEntity(entityId)
	if err != nil {
		writeError(w, err)
		return
	}

	if err = entity.Act(actParam); err != nil {
		writeError(w, err)
		return
	}

	if err = entityStore.Update(entityId, entity); err != nil {
		writeError(w, err)
		return
	}

	if entity.IsActive() {
		s.set(s.getUserId(), entityId, entity)
	} else {
		s.clear()
	}
	respJson := getEntityChangeResp(entity)
	respJson[_VERSION] = entity.GetVersion()
	writeJson(w, &respJson)
}

type session struct{
	writer http.ResponseWriter
	request *http.Request
	internalSession *sessions.Session
	userId string
	entityId string
	entity Entity
}

func getSession(w http.ResponseWriter, r *http.Request) (session, error) {
	s, err := sessionStore.Get(r, sessionName)

	session := &session{
		writer: w,
		request: r,
		internalSession: s,
	}

	var val interface{}
	var exists bool

	if val, exists = s.Values[_USER_ID]; exists {
		session.userId = val.(string)
	}else{
		session.userId = ``
	}

	if val, exists = s.Values[_ENTITY_ID]; exists {
		session.entityId = val.(string)
	}else{
		session.entityId = ``
	}

	if val, exists = s.Values[_ENTITY]; exists && val != nil {
		session.entity = val.(Entity)
	}else{
		session.entity = nil
	}

	return session, err
}

func (s *session) set(userId string, entityId string, entity Entity) error {
	s.userId = userId
	s.entityId = entityId
	s.entity = entity
	s.internalSession.Values = map[interface{}]interface{}{
		_USER_ID: userId,
		_ENTITY_ID: entityId,
		_ENTITY: entity,
	}
	return sessions.Save(s.request, s.writer)
}

func (s *session) clear() error {
	s.userId = ``
	s.entityId = ``
	s.entity = nil
	s.internalSession.Values = map[interface{}]interface{}{}
	return sessions.Save(s.request, s.writer)
}

func (s *session) isNotEngaged() bool {
	return s.entity == nil || !s.entity.IsActive()
}

func (s *session) getUserId() string {
	return s.userId
}

func (s *session) getEntityId() string {
	return s.entityId
}

func (s *session) getEntity() Entity {
	return s.entity
}

type json map[string]interface{}

func writeJson(w http.ResponseWriter, obj interface{}) error{
	js, err := js.Marshal(obj)
	w.Header().Set(`Content-Type`, `application/json`)
	w.Write(js)
	return err
}

func readJson(r *http.Request, obj interface{}) error{
	decoder := js.NewDecoder(r.Body)
	err := decoder.Decode(obj)
	return err
}

func writeError(w http.ResponseWriter, err error){
	http.Error(w, err.Error(), 500)
}

func getRequestData(r *http.Request, isForPoll bool) (entityId string, version int, err error) {
	reqJson := json{}
	readJson(r, &reqJson)
	if idParam, exists := reqJson[_ID]; exists {
		if id, ok := idParam.(string); ok {
			entityId = id
			if isForPoll {
				if versionParam, exists := reqJson[_VERSION]; exists {
					if v, ok := versionParam.(float64); ok {
						version = int(v)
					} else {
						err = errors.New(_VERSION + ` must be a number value`)
					}
				} else {
					err = errors.New(_VERSION + ` value must be included in request`)
				}
			}
		} else {
			err = errors.New(_ID + ` must be a string value`)
		}
	} else {
		err = errors.New(_ID +` value must be included in request`)
	}
	return
}

func fetchEntity(entityId string) (entity Entity, err error) {
	entity, err = entityStore.Read(entityId)
	if err == nil {
		if entity.Kick() {
			entityStore.Update(entityId, entity)
		}
	}
	return
}
