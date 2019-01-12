package handlers

import (
	"couchsport/api/models"
	"couchsport/api/stores"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/ioutil"

	"net/http"
	"strconv"
)

type userHandler struct {
	Store *stores.StoreFactory
}

//All returns all the users
func (me userHandler) All(w http.ResponseWriter, r *http.Request) {

	keys := r.URL.Query()

	users, err := me.Store.UserStore().All(keys)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	json, err := json.Marshal(users)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	fmt.Fprintf(w, string(json))

}

//Signin create a user account
func (me userHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	r.Close = true

	if r.Body != nil {
		defer r.Body.Close()
	}

	user, err := me.parseBody(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("could not parse body %s", err).Error(), http.StatusBadRequest)
		return
	}

	user, err = me.Store.UserStore().New(user)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("could not signup : %s", err).Error(), http.StatusForbidden)
		return
	}

	json, err := json.Marshal(user)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("could not marshal object %s", err).Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(json))
}

//Login authenticate the user
func (me userHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Close = true

	if r.Body != nil {
		defer r.Body.Close()
	}

	user, err := me.parseBody(r.Body)
	if err != nil {
		http.Error(w, fmt.Errorf("could not extract session %s", err).Error(), http.StatusBadRequest)
		return
	}

	dbUser, err := me.Store.UserStore().GetByEmail(user.Email)
	if err != nil {
		http.Error(w, fmt.Errorf("could not fetch user %s", err).Error(), http.StatusBadRequest)
		return
	}

	if r := comparePasswords(dbUser.Password, []byte(user.Password)); !r {
		http.Error(w, fmt.Errorf("wrong credentials").Error(), http.StatusUnauthorized)
		return
	}

	isLogged, err := me.Store.SessionStore().Create(dbUser.ID)
	if err != nil {
		http.Error(w, fmt.Errorf("could not create session %s", err).Error(), http.StatusInternalServerError)
		return
	}

	if isLogged == false {
		http.Error(w, fmt.Errorf("could not log in").Error(), http.StatusUnauthorized)
		return
	}

	cookie, err := me.Store.SessionStore().CreateCookie()

	if err != nil {
		http.Error(w, fmt.Errorf("could not create cookie %s", err).Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, cookie)

	type res struct {
		Token string
		Email string
	}

	responseBody := res{Token: me.Store.SessionStore().GetToken(), Email: dbUser.Email}

	json, err := json.Marshal(responseBody)

	if err != nil {
		http.Error(w, fmt.Errorf("could not marshal object %s", err).Error(), http.StatusInternalServerError)
	}

	fmt.Fprintf(w, string(json))
}

//IsLogged is a middleware used to know if user is Logged
func (me userHandler) IsLogged(pass func(userID uint, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		session, err := me.Store.SessionStore().GetSession(r)
		if err != nil {
			http.Error(w, fmt.Errorf("could not extract session %s", err).Error(), http.StatusUnauthorized)
			return
		}

		if session.HasExpired() {
			me.Store.SessionStore().Destroy(r)

			if err != nil {
				http.Error(w, fmt.Errorf("could not logout properly %s", err).Error(), http.StatusInternalServerError)
				return
			}

			http.Error(w, fmt.Errorf("session has expired").Error(), http.StatusUnauthorized)
			return
		}

		pass(session.OwnerID, w, r)
	}
}

//Logout log out the user
func (me userHandler) Logout(_ uint, w http.ResponseWriter, r *http.Request) {
	success, err := me.Store.SessionStore().Destroy(r)
	if err != nil {
		http.Error(w, fmt.Errorf("could not logout properly %s", err).Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, `{ "Result" : `+strconv.FormatBool(success)+` }`)
}

func comparePasswords(hashedPwd string, plainPwd []byte) bool {
	// Since we'll be getting the hashed password from the DB it
	// will be a string so we'll need to convert it to a byte slice
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		return false
	}

	return true
}

func (me userHandler) parseBody(body io.Reader) (models.User, error) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return models.User{}, err
	}

	var u models.User
	err = json.Unmarshal(b, &u)

	if err != nil {
		return models.User{}, err
	}

	return u, nil
}