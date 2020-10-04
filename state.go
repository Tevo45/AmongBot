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

type persistentState struct {
	Premium premiumMemberships	`format:toml location:premium.toml`
	GuildPrefs map[string]*guildPrefs	`directory location:guild-props`
}

// TODO Automate this
type premiumMemberships struct {
	Servers []string
}

type guildPrefs struct {
	PlayChan string
}

func (s *persistentState) GetPremiumGuilds() []string {
	return s.Premium.Servers
}

var state = persistentState{}

// Maybe our persistent state management is a little bit too complicated

func loadState() (errs error) {
	if fi, e := os.Stat("state"); os.IsNotExist(e) {
		fmt.Println("No state database found, generating new one")
		err := initState()
		if err != nil {
			errs = coalesce(errs, err)
			return
		}
		err= saveState()
		if err != nil {
			errs = coalesce(errs, err)
		}
		return
	} else if !fi.Mode().IsDir() {
		// FIXME Make logging across codebase consistent, use log package instead
		fmt.Println(
			"***WARNING*** file 'state' exists but is not a directory, persistent state will not work!")
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
		err := tryLoadField(absStruct.Field(c), concStruct.Field(c))
		if err != nil {
			errs = coalesce(errs, err)
		}
	}
	return
}

func initState() (err error) {
	err = os.Mkdir("state", 0644)
	return
}

func saveState() (errs error) {
	stStruct := reflect.TypeOf(state)
	cStruct := reflect.ValueOf(state)
	for c := 0; c < stStruct.NumField(); c++ {
		err := saveField(stStruct.Field(c), cStruct.Field(c))
		if err != nil {
			errs = coalesce(errs, err)
		}
	}
	return
}

// FIXME Much repeated code between functions
// We probably can get away with only passing a reflect.Value around

func saveField(absField reflect.StructField, concField reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Caught panic at saveField: %v", r)
		}
	}()
	opts := tagToMap(string(absField.Tag))
	if _, ok := opts["directory"]; ok {
		err = saveDirectory(absField, concField)
		return
	}
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
	switch format {
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

func saveDirectory(absField reflect.StructField, concField reflect.Value) (err error) {
	// We already parsed this once on saveField(), but I don't think it matters that much
	opts := tagToMap(string(absField.Tag))
	format := "json"
	if f := opts["format"]; f != nil {
		format = *f
	}
	location := absField.Name + "." + format
	if l := opts["location"]; l != nil {
		location = *l
	}
	// TODO We should serialize structs as well, maybe arrays
	if fi, e := os.Stat("state"); os.IsNotExist(e) {
		err = os.Mkdir("state/"+location, 0644)
		if err != nil {
			return
		}
	} else if !fi.Mode().IsDir() {
		err = errors.New("File exists and is not directory")
		return
	}
	if concField.Kind() != reflect.Map {
		err = errors.New("Sorry bud, only maps allowed")
		return
	}
	for _, k := range concField.MapKeys() {
		fname := fmt.Sprintf("state/%s/%v.%s", location, k, format)
		var buf []byte
		switch format {
		case "toml":
			buf, err = toml.Marshal(concField.MapIndex(k).Interface())
		case "json":
			buf, err = json.Marshal(concField.MapIndex(k).Interface())
		default:
			err = errors.New("Unsupported format: " + format)
			return
		}
		e := ioutil.WriteFile(fname, buf, 0644)
		if e != nil {
			err = coalesce(err, e)
		}
	}
	return
}

func tryLoadField(absField reflect.StructField, concField reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Caught panic at tryLoadField: %v", r)
		}
	}()
	if !concField.CanSet() {
		err = errors.New("Field is not settable")
		return
	}
	defer func() {
		if r := recover(); r != nil {
			r = err
		}
	}()
	opts := tagToMap(string(absField.Tag))
	if _, ok := opts["directory"]; ok {
		err = tryLoadDirectory(absField, concField)
		return
	}
	format := "json"
	if f := opts["format"]; f != nil {
		format = *f
	}
	location := absField.Name + "." + format
	if l := opts["location"]; l != nil {
		location = *l
	}
	buf, err := ioutil.ReadFile("state/" + location)
	if err != nil {
		return
	}
	switch format {
	case "toml":
		err = toml.Unmarshal(buf, concField.Addr().Interface())
		if err != nil {
			return
		}
	case "json":
		err = toml.Unmarshal(buf, concField.Addr().Interface())
		if err != nil {
			return
		}
	default:
		err = errors.New("Unsupported format: " + format)
		return
	}
	return
}

func tryLoadDirectory(absField reflect.StructField, concField reflect.Value) (err error) {
	opts := tagToMap(string(absField.Tag))
	format := "json"
	if f := opts["format"]; f != nil {
		format = *f
	}
	location := absField.Name + "." + format
	if l := opts["location"]; l != nil {
		location = *l
	}
	if fi, e := os.Stat("state"); os.IsNotExist(e) {
		err = e
		return
	} else if !fi.Mode().IsDir() {
		err = errors.New("File exists and is not directory")
		return
	}
	if concField.Kind() != reflect.Map {
		err = errors.New("Sorry bud, only maps allowed")
		return
	}
	loop:
	for _, k := range concField.MapKeys() {
		fname := fmt.Sprintf("state/%s/%v.%s", location, k, format)
		buf, e := ioutil.ReadFile("state/" + fname)
		if e != nil {
			err = coalesce(err, e)
			continue loop
		}
		switch format {
		case "toml":
			e := toml.Unmarshal(buf, concField.MapIndex(k).Addr().Interface())
			if e != nil {
				err = coalesce(err, e)
				continue loop
			}
		case "json":
			e := json.Unmarshal(buf, concField.MapIndex(k).Addr().Interface())
			if e != nil {
				err = coalesce(err, e)
				continue loop
			}
		default:
			err = coalesce(err, errors.New("Unsupported format: " + format))
			return
		}
	}
	return
}
