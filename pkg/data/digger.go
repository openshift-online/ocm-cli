/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains the implementation of an object that knows how to extract data from objects
// using paths.

package data

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// DiggerBuilder contains the information and logic needed to build a digger.
type DiggerBuilder struct {
}

// Digger is an object that knows how to extract information from objects using paths.
type Digger struct {
	methodCache     map[cacheKey]reflect.Method
	methodCacheLock *sync.Mutex
	fieldCache      map[cacheKey]int
	fieldCacheLock  *sync.Mutex
}

// cacheKey is used as the key for the methods and fields caches.
type cacheKey struct {
	class reflect.Type
	field string
}

// NewDigger creates a builder that can then be used to configure and create diggers.
func NewDigger() *DiggerBuilder {
	return &DiggerBuilder{}
}

// Build uses the configuration stored in the builder to create a new digger.
func (b *DiggerBuilder) Build(ctx context.Context) (result *Digger, err error) {
	// Create and populate the object:
	result = &Digger{
		methodCache:     map[cacheKey]reflect.Method{},
		methodCacheLock: &sync.Mutex{},
		fieldCache:      map[cacheKey]int{},
		fieldCacheLock:  &sync.Mutex{},
	}

	return
}

// Dig extracts from the given object the field that corresponds to the given path. The path should
// be a sequence of field names separated by dots.
func (d *Digger) Dig(object interface{}, path string) interface{} {
	path = strings.TrimSpace(path)
	if path == "" {
		return object
	}
	segments := strings.Split(path, ".")
	if len(segments) == 0 {
		return object
	}
	for i, field := range segments {
		segments[i] = strings.TrimSpace(field)
	}
	return d.digPath(object, segments)
}

func (d *Digger) digPath(object interface{}, path []string) interface{} {
	if len(path) == 0 {
		return object
	}
	head := path[0]
	next := d.digField(object, head)
	if next == nil {
		return nil
	}
	tail := path[1:]
	return d.digPath(next, tail)
}

func (d *Digger) digField(object interface{}, field string) interface{} {
	value := reflect.ValueOf(object)
	return d.digFieldFromValue(value, field)
}

func (d *Digger) digFieldFromValue(value reflect.Value, field string) interface{} {
	switch value.Kind() {
	case reflect.Ptr:
		return d.digFieldFromPtr(value, field)
	case reflect.Struct:
		return d.digFieldFromStruct(value, field)
	default:
		return nil
	}
}

func (d *Digger) digFieldFromPtr(value reflect.Value, name string) interface{} {
	// Try to find a matching method:
	method, ok := d.lookupMethod(value.Type(), name)
	if ok {
		return d.digFieldFromMethod(value, method)
	}

	// If no matching method was found, but the target of the pointer is a struct then we should
	// try to extract the field from the public methods of the struct.
	if value.Type().Elem().Kind() == reflect.Struct {
		return d.digFieldFromStruct(value.Elem(), name)
	}

	// If we are here we didn't find any match:
	return nil
}

func (d *Digger) digFieldFromStruct(value reflect.Value, name string) interface{} {
	// First try to find a matching method:
	method, ok := d.lookupMethod(value.Type(), name)
	if ok {
		return d.digFieldFromMethod(value, method)
	}

	// If no matching method was found, try to find a matching public field:
	index, ok := d.lookupField(value.Type(), name)
	if ok {
		return value.Field(index).Interface()
	}

	// If we are here we didn't find any match:
	return nil
}

func (d *Digger) digFieldFromMethod(value reflect.Value, method reflect.Method) interface{} {
	var result reflect.Value

	// Call the method:
	inArgs := []reflect.Value{
		value,
	}
	outArgs := method.Func.Call(inArgs)

	// If the method has one output parameter then we assume it is the value. If it has two
	// output parameters then we assume that the first is the value and the second is a boolean
	// flag indicating if there is actually a value.
	switch len(outArgs) {
	case 1:
		result = outArgs[0]
	case 2:
		if outArgs[1].Bool() {
			result = outArgs[0]
		}
	}

	// Return the result:
	if !result.IsValid() {
		return nil
	}
	return result.Interface()
}

// lookupMethod tries to find a method of the given value that matches the given path segment. For
// example, if the path segment is `my_field` it will look for a method named `GetMyField` or
// `MyField`. Only methods that don't have input parameters will be considered.
func (d *Digger) lookupMethod(class reflect.Type, field string) (result reflect.Method, ok bool) {
	// Acquire the method cache lock:
	d.methodCacheLock.Lock()
	defer d.methodCacheLock.Unlock()

	// Try to find the method in the cache:
	key := cacheKey{
		class: class,
		field: field,
	}
	result, ok = d.methodCache[key]
	if ok {
		return
	}

	// Get the number of methods:
	count := class.NumMethod()

	// Try to find a method that returns a value and a boolean flag indicating if there is
	// actually a value. We try this first because this gives more information and allows us to
	// return nil when the field isn't present, instead of returning the zero value of the type.
	for i := 0; i < count; i++ {
		method := class.Method(i)
		if method.Type.NumIn() != 1 || method.Type.NumOut() != 2 {
			continue
		}
		if method.Type.Out(1).Kind() != reflect.Bool {
			continue
		}
		if !d.methodNameMatches(method.Name, field) {
			continue
		}
		d.methodCache[key] = method
		result = method
		ok = true
		return
	}

	// Try now to find a method that returns only the value.
	for i := 0; i < count; i++ {
		method := class.Method(i)
		if method.Type.NumIn() != 1 || method.Type.NumOut() != 1 {
			continue
		}
		if !d.methodNameMatches(method.Name, field) {
			continue
		}
		d.methodCache[key] = method
		result = method
		ok = true
		return
	}

	// If we are here then we didn't find any matching method.
	ok = false
	return
}

// methodNameMatches checks if the name of a Go method matches a path segment.
func (d *Digger) methodNameMatches(method, segment string) bool {
	// If there is a `Get` prefix remove it:
	name := method
	if getMethodRE.MatchString(method) {
		name = name[3:]
	}

	// Check if the method name matches the segment:
	return d.nameMatches(name, segment)
}

// lookupField tries to find a field of the given value that matches the given path segment. For
// example, if the path segment is `my_field` it will look for a field named `MyField`.
func (d *Digger) lookupField(class reflect.Type, name string) (result int, ok bool) {
	// Acquire the field cache lock:
	d.fieldCacheLock.Lock()
	defer d.fieldCacheLock.Unlock()

	// Try to find the field in the cache:
	key := cacheKey{
		class: class,
		field: name,
	}
	result, ok = d.fieldCache[key]
	if ok {
		return
	}

	// Try now to find a field that matches the name:
	count := class.NumField()
	for i := 0; i < count; i++ {
		field := class.Field(i)
		if !d.fieldNameMatches(field.Name, name) {
			continue
		}
		d.fieldCache[key] = i
		result = i
		ok = true
		return
	}

	// If we are here then we didn't find any matching method.
	ok = false
	return
}

// fieldNameMatches checks if the name of a Go fields matches a path segment.
func (d *Digger) fieldNameMatches(field, segment string) bool {
	return d.nameMatches(field, segment)
}

// nameMatches checks if a Go name matches with a path segment.
func (d *Digger) nameMatches(name, segment string) bool {
	// Conver the strings to arrays of runes so that we can compare runes one by one easily:
	nameRunes := []rune(name)
	nameLen := len(nameRunes)
	segmentRunes := []rune(segment)
	segmentLen := len(segmentRunes)

	// Start at the beginning of both arrays of runes, and advance while the runes in both the
	// name and the path segment are compatible. Two runes are compatible if they are equal
	// ignoring case. An underscore in the path segment is compatible if there is a transition
	// from lower case to upper case in the name at that point.
	nameI, segmentI := 0, 0
	for nameI < nameLen && segmentI < segmentLen {
		if unicode.ToLower(nameRunes[nameI]) == unicode.ToLower(segmentRunes[segmentI]) {
			nameI++
			segmentI++
			continue
		}
		if nameI > 0 && segmentRunes[segmentI] == '_' {
			previousLower := unicode.IsLower(nameRunes[nameI-1])
			currentUpper := unicode.IsUpper(nameRunes[nameI])
			if previousLower && currentUpper {
				segmentI++
				continue
			}
		}
		return false
	}

	// If we have consumed all the runes of both names then there is a match:
	return nameI == nameLen && segmentI == segmentLen
}

// getMethodRE is a regular expression used to check if a method name starts with `Get`. Note that
// checking if the string starts with `Get` is not enough as it would fails for methods with names
// like `Getaway`.
var getMethodRE = regexp.MustCompile(`^Get\p{Lu}`)
