package handlers

import (
	"couchsport/api/models"
	"couchsport/api/stores"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
)

type profileHandler struct {
	Store *stores.StoreFactory
}

//Mine returns the connected user profile
func (me profileHandler) Mine(UserID uint, w http.ResponseWriter, r *http.Request) {
	profile, err := me.Store.ProfileStore().GetProfileByOwnerID(UserID)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	json, err := json.Marshal(profile)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(json))

}

//Update the user profile
func (me profileHandler) Update(userID uint, w http.ResponseWriter, r *http.Request) {
	r.Close = true

	if r.Body != nil {
		defer r.Body.Close()
	}

	profile, err := me.parseBody(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	owns, err := me.Store.AuthorizationStore().OwnProfile(userID, profile.ID)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	if !owns {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusForbidden)
		return
	}

	profile, err = me.Store.ProfileStore().Update(profile.ID, profile)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusBadRequest)
		return
	}

	json, err := json.Marshal(profile)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("could encode output %s", err).Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, string(json))

}

func (me profileHandler) parseBody(body io.Reader) (models.Profile, error) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return models.Profile{}, err
	}

	var obj models.Profile
	err = json.Unmarshal(b, &obj)

	if err != nil {
		return models.Profile{}, err
	}

	return obj, nil
}
