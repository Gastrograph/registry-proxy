package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

func registry(w http.ResponseWriter, r *http.Request) {
	director := func(req *http.Request) {
		reverse := REGISTRY_HOST + ":" + REGISTRY_PORT
		req = r
		req.URL.Scheme = "http"
		req.URL.Host = reverse
	}
	roundTripper := &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   900 * time.Second,
			KeepAlive: 65 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: roundTripper,
	}
	proxy.ServeHTTP(w, r)
}
func (c *AppContext) loginHandler(w http.ResponseWriter, r *http.Request) {
	ctx, ok := context.GetAllOk(r)
	if !ok {
		return
	}
	yield.HTML(w, http.StatusOK, "login", ctx, c.layout)
}
func (c *AppContext) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if r.Form["username"] == nil {
		http.Redirect(w, r, "/login", 302)
		return
	}
	username := r.Form["username"][0]
	if r.Form["password"] == nil {
		http.Redirect(w, r, "/login", 302)
		return
	}
	password := r.Form["password"][0]
	match, err := c.Login(username, password)
	if err != nil {
		log.Println(err)
		return
		//TODO: put actual 500
	}
	if match {
		session, err := store.Get(r, "registry")
		if err != nil {
			log.Println(err)
			return
		}
		session.Options = &sessions.Options{
			MaxAge: 86400 * 7,
		}
		session.Values["username"] = username
		session.Save(r, w)
		http.Redirect(w, r, "/ui", 302)
		return
	}
	http.Redirect(w, r, "/login", 302)
}
func (c *AppContext) ui(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("http://%s:%s/v2/_catalog", REGISTRY_HOST, REGISTRY_PORT)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return
	}
	if resp.StatusCode != 200 {
		log.Println(resp.Status)
		str, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		log.Println(string(str))
	}
	var catalog Catalog
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&catalog)
	if err != nil {
		return
	}
	context.Set(r, "repositories", catalog.Repositories)
	ctx, ok := context.GetAllOk(r)
	if !ok {
		return
	}
	yield.HTML(w, http.StatusOK, "index", ctx, c.layout)
}
func (c *AppContext) repoShow(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	url := fmt.Sprintf("http://%s:%s/v2/%s/tags/list",
		REGISTRY_HOST,
		REGISTRY_PORT,
		name)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return
	}
	if resp.StatusCode != 200 {
		log.Println(resp.Status)
		str, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		log.Println(string(str))
	}
	var tagList TagList
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&tagList)
	if err != nil {
		return
	}
	context.Set(r, "tagList", tagList)
	ctx, ok := context.GetAllOk(r)
	if !ok {
		return
	}
	yield.HTML(w, http.StatusOK, "show", ctx, c.layout)
}