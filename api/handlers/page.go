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

type pageHandler struct {
	Store *stores.StoreFactory
}

//All return all the pages
func (me pageHandler) All(w http.ResponseWriter, r *http.Request) {
	pages, err := me.Store.PageStore().All(r.URL.Query())
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	json, err := json.Marshal(pages)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(json))

}

//ProfilePages gets all the profile pages
func (me pageHandler) ProfilePages(userID uint, w http.ResponseWriter, r *http.Request) {
	r.Close = true

	if r.Body != nil {
		defer r.Body.Close()
	}

	profileID, err := me.Store.AuthorizationStore().GetProfileID(userID)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	pages, err := me.Store.PageStore().GetPagesByOwnerID(profileID)
	if err != nil {
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	json, err := json.Marshal(pages)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(json))

}

//New creates a new Page in database
func (me pageHandler) New(userID uint, w http.ResponseWriter, r *http.Request) {
	r.Close = true

	if r.Body != nil {
		defer r.Body.Close()
	}

	page, err := me.parseBody(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	profileID, err := me.Store.AuthorizationStore().GetProfileID(userID)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	pageObj, err := me.Store.PageStore().New(profileID, page)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("could not create page %s", err).Error(), http.StatusBadRequest)
		return
	}

	json, err := json.Marshal(pageObj)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(json))

}

//Update the current page
func (me pageHandler) Update(userID uint, w http.ResponseWriter, r *http.Request) {
	r.Close = true

	if r.Body != nil {
		defer r.Body.Close()
	}

	page, err := me.parseBody(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	if !page.IsValid("UPDATE") {
		http.Error(w, fmt.Errorf("Page is invalid").Error(), http.StatusUnprocessableEntity)
		return
	}

	owns, err := me.Store.AuthorizationStore().OwnPage(userID, page.ID)
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

	pageObj, err := me.Store.PageStore().Update(userID, page)
	if err != nil {
		http.Error(w, fmt.Errorf("could not update page %s", err).Error(), http.StatusBadRequest)
		return
	}

	json, err := json.Marshal(pageObj)

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(json))
}

//Delete the page
func (me pageHandler) Delete(userID uint, w http.ResponseWriter, r *http.Request) {
	r.Close = true

	if r.Body != nil {
		defer r.Body.Close()
	}

	page, err := me.parseBody(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	owns, err := me.Store.AuthorizationStore().OwnPage(userID, page.ID)
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

	result, err := me.Store.PageStore().Delete(userID, page.ID)
	if err != nil {
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusBadRequest)
		return
	}

	json, err := json.Marshal(struct{ Result bool }{Result: result})

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(json))
}

//Public the page setting Public to 1 or 0
func (me pageHandler) Publish(userID uint, w http.ResponseWriter, r *http.Request) {
	r.Close = true

	if r.Body != nil {
		defer r.Body.Close()
	}

	page, err := me.parseBody(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusUnprocessableEntity)
		return
	}

	owns, err := me.Store.AuthorizationStore().OwnPage(userID, page.ID)
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

	result, err := me.Store.PageStore().Publish(userID, page.ID, page.Public)
	if err != nil {
		http.Error(w, fmt.Errorf("could not extract body %s", err).Error(), http.StatusBadRequest)
		return
	}

	json, err := json.Marshal(struct{ Result bool }{Result: result})

	if err != nil {
		log.Error(err)
		http.Error(w, fmt.Errorf("%s", err).Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(json))
}

func (me pageHandler) parseBody(body io.Reader) (models.Page, error) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return models.Page{}, err
	}

	var obj models.Page
	err = json.Unmarshal(b, &obj)

	if err != nil {
		return models.Page{}, err
	}

	return obj, nil
}
