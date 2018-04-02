/*
 * DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
 *
 * Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved.
 *
 * The contents of this file are subject to the terms of either the GNU
 * General Public License Version 2 only ("GPL") or the Common Development
 * and Distribution License("CDDL") (collectively, the "License").  You
 * may not use this file except in compliance with the License.  You can
 * obtain a copy of the License at
 * https://glassfish.dev.java.net/public/CDDL+GPL_1_1.html
 * or packager/legal/LICENSE.txt.  See the License for the specific
 * language governing permissions and limitations under the License.
 *
 * When distributing the software, include this License Header Notice in each
 * file and include the License file at packager/legal/LICENSE.txt.
 *
 * GPL Classpath Exception:
 * Oracle designates this particular file as subject to the "Classpath"
 * exception as provided by Oracle in the GPL Version 2 section of the License
 * file that accompanied this code.
 *
 * Modifications:
 * If applicable, add the following below the License Header, with the fields
 * enclosed by brackets [] replaced by your own identifying information:
 * "Portions Copyright [year] [name of copyright owner]"
 *
 * Contributor(s):
 * If you wish your version of this file to be governed by only the CDDL or
 * only the GPL Version 2, indicate your decision by adding "[Contributor]
 * elects to include this software in this distribution under the [CDDL or GPL
 * Version 2] license."  If you don't indicate a single choice of license, a
 * recipient has the option to distribute your version of this file under
 * either the CDDL, the GPL Version 2 or to extend the choice of license to
 * its licensees as provided above.  However, if you add GPL Version 2 code
 * and therefore, elected the GPL Version 2 license, then the option applies
 * only if the new code is made subject to such option by the copyright
 * holder.
 */

package utilities

import (
	"errors"
	"fmt"
	"github.com/jwells131313/goethe"
	"github.com/jwells131313/goethe/internal"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type goetheData struct {
	tidMux       sync.Mutex
	lastTid      int64
	poolMap      map[string]goethe.Pool
	threadLocals map[string]*threadLocalOperators
}

type threadLocalOperators struct {
	initializer func() interface{}
	destroyer   func(interface{})
	lock        goethe.Lock
	actuals     map[int64]interface{}
}

var (
	errorType    = reflect.TypeOf(errors.New("")).String()
	globalGoethe = newGoethe()
)

func newGoethe() *goetheData {
	retVal := &goetheData{
		lastTid:      9,
		poolMap:      make(map[string]goethe.Pool),
		threadLocals: make(map[string]*threadLocalOperators),
	}

	return retVal
}

// GetGoethe returns the systems goth global
func GetGoethe() goethe.Goethe {
	return globalGoethe
}

func (goth *goetheData) getAndIncrementTid() int64 {
	goth.tidMux.Lock()
	defer goth.tidMux.Unlock()

	goth.lastTid++
	return goth.lastTid
}

func (goth *goetheData) Go(userCall func()) (int64, error) {
	return goth.GoWithArgs(userCall)
}

func (goth *goetheData) GoWithArgs(userCall interface{}, args ...interface{}) (int64, error) {
	tid := goth.getAndIncrementTid()

	// do some type checking
	typ := reflect.TypeOf(userCall)
	kin := typ.Kind()
	if kin != reflect.Func {
		return tid, fmt.Errorf("first argument of GoWithArgs must be a function, it is %s", kin.String())
	}

	arguments := make([]reflect.Value, 0)
	for _, arg := range args {
		argValue := reflect.ValueOf(arg)

		arguments = append(arguments, argValue)

		// TODO:  Here it would be nice to type check the parameter value
	}

	go invokeStart(tid, userCall, arguments)

	return tid, nil
}

func (goth *goetheData) GetThreadID() int64 {
	stackAsBytes := debug.Stack()
	stackAsString := string(stackAsBytes)

	tokenized := strings.Split(stackAsString, "xXTidFrame")

	var tidHexString string
	first := true
	gotOne := false
	for _, tok := range tokenized {
		if first {
			first = false
		} else {
			gotOne = true
			tidHexString = string(tok[0]) + tidHexString
		}
	}

	if !gotOne {
		return -1
	}

	var result int

	fmt.Sscanf(tidHexString, "%X", &result)

	return int64(result)
}

func (goth *goetheData) NewGoetheLock() goethe.Lock {
	return internal.NewReaderWriterLock(goth)
}

// NewBoundedFunctionQueue returns a function queue with the given capacity
func (goth *goetheData) NewBoundedFunctionQueue(capacity uint32) goethe.FunctionQueue {
	return internal.NewFunctionQueue(capacity)
}

// NewErrorQueue returns an error queue with the given capacity.  If errors
// are returned when the ErrorQueue is at capacity the new errors are droppedmin
func (goth *goetheData) NewErrorQueue(capacity uint32) goethe.ErrorQueue {
	return internal.NewBoundedErrorQueue(capacity)
}

// NewPool is the native implementation of NewPool
func (goth *goetheData) NewPool(name string, minThreads int32, maxThreads int32, idleDecayDuration time.Duration,
	functionQueue goethe.FunctionQueue, errorQueue goethe.ErrorQueue) (goethe.Pool, error) {
	goth.tidMux.Lock()
	defer goth.tidMux.Unlock()

	foundPool, found := goth.poolMap[name]
	if found {
		return foundPool, goethe.ErrPoolAlreadyExists
	}

	retVal, err := NewThreadPool(goth, name, minThreads, maxThreads, idleDecayDuration, functionQueue,
		errorQueue)
	if err != nil {
		return nil, err
	}

	goth.poolMap[name] = retVal

	return retVal, nil
}

// GetPool returns a non-closed pool with the given name.  If not found second
// value returned will be false
func (goth *goetheData) GetPool(name string) (goethe.Pool, bool) {
	goth.tidMux.Lock()
	goth.tidMux.Unlock()

	retVal, found := goth.poolMap[name]

	return retVal, found
}

// EstablishThreadLocal tells the system of the named thread local storage
// initialize method and destroy method.  This method can be called on any
// thread, including non-goethe threads
func (goth *goetheData) EstablishThreadLocal(name string, initializer func() interface{},
	destroyer func(interface{})) error {
	goth.tidMux.Lock()
	goth.tidMux.Unlock()

	_, found := goth.threadLocals[name]
	if found {
		return fmt.Errorf("There is already an established thread local for %s", name)
	}

	operation := &threadLocalOperators{
		initializer: initializer,
		destroyer:   destroyer,
		lock:        goth.NewGoetheLock(),
		actuals:     make(map[int64]interface{}),
	}

	goth.threadLocals[name] = operation

	return nil
}

// Get thread local returns the instance of the storage associated with
// the current goethe thread.  May only be called on goethe threads and
// will return ErrNotGoetheThread if called from a non-goethe thread.
// If EstablishThreadLocal with the given name has not been called prior to
// this function call then ErrNoThreadLocalEstablished will be returned
func (goth *goetheData) GetThreadLocal(name string) (interface{}, error) {
	tid := goth.GetThreadID()
	if tid < int64(0) {
		return nil, goethe.ErrNotGoetheThread
	}

	operators, found := goth.getOperatorsByName(name)
	if !found {
		return nil, goethe.ErrNoThreadLocalEstablished
	}

	operators.lock.WriteLock()
	defer operators.lock.WriteUnlock()

	actual, found := operators.actuals[tid]
	if !found {
		actual = operators.initializer()

		operators.actuals[tid] = actual
	}

	return actual, nil
}

func (goth *goetheData) getOperatorsByName(name string) (*threadLocalOperators, bool) {
	goth.tidMux.Lock()
	goth.tidMux.Unlock()

	retVal, found := goth.threadLocals[name]

	return retVal, found
}

func removeThreadLocal(operators *threadLocalOperators, tid int64) {
	operators.lock.WriteLock()
	defer operators.lock.WriteUnlock()

	actual, found := operators.actuals[tid]
	if !found {
		return
	}

	if operators.destroyer != nil {
		operators.destroyer(actual)
	}

	delete(operators.actuals, tid)
}

func (goth *goetheData) removeAllActuals(tid int64) {
	goth.tidMux.Lock()
	goth.tidMux.Unlock()

	for _, operators := range goth.threadLocals {
		removeThreadLocal(operators, tid)
	}
}

func (goth *goetheData) removePool(name string) {
	goth.tidMux.Lock()
	goth.tidMux.Unlock()

	delete(goth.poolMap, name)
}

// convertToNibbles returns the nibbles of the string
func convertToNibbles(tid int64) []byte {
	if tid < 0 {
		panic("The tid must not be negative")
	}

	asString := fmt.Sprintf("%x", tid)
	return []byte(asString)
}

func invokeStart(tid int64, userCall interface{}, args []reflect.Value) error {
	nibbles := convertToNibbles(tid)

	return internalInvoke(tid, 0, nibbles, userCall, args)
}

func invokeEnd(tid int64, userCall interface{}, args []reflect.Value) error {
	defer globalGoethe.removeAllActuals(tid)

	val := reflect.ValueOf(userCall)
	retVals := val.Call(args)

	if len(retVals) > 0 {
		rVal := retVals[0]
		if rVal.IsNil() {
			return nil
		}

		if !rVal.CanInterface() {
			return nil
		}

		if rVal.Type().String() == "error" {
			iFace := rVal.Interface()
			retVal := iFace.(error)
			return retVal
		}
	}

	return nil
}

func internalInvoke(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	if index >= len(nibbles) {
		return invokeEnd(tid, userCall, args)
	}

	currentFrame := nibbles[index]
	switch currentFrame {
	case byte('0'):
		return xXTidFrame0(tid, index, nibbles, userCall, args)
	case byte('1'):
		return xXTidFrame1(tid, index, nibbles, userCall, args)
	case byte('2'):
		return xXTidFrame2(tid, index, nibbles, userCall, args)
	case byte('3'):
		return xXTidFrame3(tid, index, nibbles, userCall, args)
	case byte('4'):
		return xXTidFrame4(tid, index, nibbles, userCall, args)
	case byte('5'):
		return xXTidFrame5(tid, index, nibbles, userCall, args)
	case byte('6'):
		return xXTidFrame6(tid, index, nibbles, userCall, args)
	case byte('7'):
		return xXTidFrame7(tid, index, nibbles, userCall, args)
	case byte('8'):
		return xXTidFrame8(tid, index, nibbles, userCall, args)
	case byte('9'):
		return xXTidFrame9(tid, index, nibbles, userCall, args)
	case byte('a'):
		return xXTidFrameA(tid, index, nibbles, userCall, args)
	case byte('b'):
		return xXTidFrameB(tid, index, nibbles, userCall, args)
	case byte('c'):
		return xXTidFrameC(tid, index, nibbles, userCall, args)
	case byte('d'):
		return xXTidFrameD(tid, index, nibbles, userCall, args)
	case byte('e'):
		return xXTidFrameE(tid, index, nibbles, userCall, args)
	case byte('f'):
		return xXTidFrameF(tid, index, nibbles, userCall, args)
	default:
		panic("unknown type")

	}

}

func xXTidFrame0(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame1(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame2(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame3(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame4(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame5(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame6(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame7(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame8(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrame9(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrameA(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrameB(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrameC(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrameD(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrameE(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}

func xXTidFrameF(tid int64, index int, nibbles []byte, userCall interface{}, args []reflect.Value) error {
	return internalInvoke(tid, index+1, nibbles, userCall, args)
}
