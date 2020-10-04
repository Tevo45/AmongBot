package main

import (
	"fmt"
	"time"
	"io/ioutil"
	"os"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/pelletier/go-toml"
)

// TODO Improve this
type persistentState struct {
	Premium premiumMemberships	`format:toml location:premium.toml`
}

// TODO Automate this
type premiumMemberships struct {
	Servers []string
}

func (s *persistentState) GetPremiumGuilds() []string {
	return []string{"758117713192288256"}
}

var state = persistentState{}

func loadState() (errs []error) {
	if _, e := os.Stat("state"); os.IsNotExist(e) {
		fmt.Println("No state database found, generating new one")
		err := initState()
		if err != nil {
			errs = append(errs, err)
			return
		}
		es := saveState()
		if es != nil {
			errs = append(errs, es...)
		}
		return
	}
	go func() {
		t := time.NewTicker(time.Minute)
		for {
			<-t.C
			err := saveState()
			if err != nil {
				fmt.Printf("Error saving while persistent state: %s\n", err)
			}
		}
	}()
	absStruct := reflect.TypeOf(state)
	concStruct := reflect.ValueOf(&state).Elem()
	for c := 0; c < absStruct.NumField(); c++ {
		err := tryLoadField(concStruct.Field(c))
		if err != nil {
			errs = append(errs, err)
		}
	}
	saveState()
	return
}

func initState() (err error) {
	err = os.Mkdir("state", 0644)
	return
}

func saveState() (errs []error) {
	stStruct := reflect.TypeOf(state)
	cStruct := reflect.ValueOf(state)
	for c := 0; c < stStruct.NumField(); c++ {
		err := saveField(stStruct.Field(c), cStruct.Field(c))
		if err != nil {
			errs = append(errs, err)
		}
	}
	return
}

func saveField(absField reflect.StructField, concField reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			r = err
		}
	}()
	opts := tagToMap(string(absField.Tag))
	format := "json"
	if f := opts["format"]; f != nil {
		format = *f
	}
	location := absField.Name + "." + format
	if l := opts["location"]; l != nil {
		location = *l
	}
	// TOML for data meant to be edited by humans, JSON otherwise
	var buf []byte
	switch(format) {
	case "toml":
		buf, err = toml.Marshal(concField.Interface())
	case "json":
		buf, err = json.Marshal(concField.Interface())
	default:
		err = errors.New("Unsupported format: " + format)
		return
	}
	if err != nil {
		return
	}
	err = ioutil.WriteFile("state/" + location, buf, 0644)
	return
}

func tryLoadField(concField reflect.Value) (err error) {
	err = errors.New("Not implemented")
	return
}
