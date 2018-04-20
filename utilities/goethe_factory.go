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

type poolData struct {
	poolMux sync.Mutex
	poolMap map[string]goethe.Pool
}

type timersData struct {
	timerMux sync.Mutex
	timer    timerImpl
}

type threadLocalsData struct {
	localsMux    sync.Mutex
	threadLocals map[string]*threadLocalOperators
}

type goetheData struct {
	tidMux  sync.Mutex
	lastTid int64

	pools  *poolData
	timers *timersData
	locals *threadLocalsData
}

type threadLocalOperators struct {
	initializer func(goethe.ThreadLocal) error
	destroyer   func(goethe.ThreadLocal) error
	lock        goethe.Lock
	actuals     map[int64]goethe.ThreadLocal
}

var (
	errorType    = reflect.TypeOf(errors.New("")).String()
	globalGoethe = newGoethe()
)

const (
	timerTid = 9
)

func newGoethe() *goetheData {
	pools := &poolData{
		poolMap: make(map[string]goethe.Pool),
	}

	timers := &timersData{}

	locals := &threadLocalsData{
		threadLocals: make(map[string]*threadLocalOperators),
	}

	retVal := &goetheData{
		lastTid: 9,
		pools:   pools,
		timers:  timers,
		locals:  locals,
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

	argArray := make([]interface{}, len(args))
	for index, arg := range args {
		argArray[index] = arg
	}

	arguments, err := GetValues(userCall, argArray)
	if err != nil {
		return -1, err
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
	goth.pools.poolMux.Lock()
	defer goth.pools.poolMux.Unlock()

	foundPool, found := goth.pools.poolMap[name]
	if found {
		return foundPool, goethe.ErrPoolAlreadyExists
	}

	retVal, err := newThreadPool(goth, name, minThreads, maxThreads, idleDecayDuration, functionQueue,
		errorQueue)
	if err != nil {
		return nil, err
	}

	goth.pools.poolMap[name] = retVal

	return retVal, nil
}

// GetPool returns a non-closed pool with the given name.  If not found second
// value returned will be false
func (goth *goetheData) GetPool(name string) (goethe.Pool, bool) {
	goth.pools.poolMux.Lock()
	goth.pools.poolMux.Unlock()

	retVal, found := goth.pools.poolMap[name]

	return retVal, found
}

// EstablishThreadLocal tells the system of the named thread local storage
// initialize method and destroy method.  This method can be called on any
// thread, including non-goethe threads
func (goth *goetheData) EstablishThreadLocal(name string, initializer func(goethe.ThreadLocal) error,
	destroyer func(goethe.ThreadLocal) error) error {
	goth.locals.localsMux.Lock()
	goth.locals.localsMux.Unlock()

	_, found := goth.locals.threadLocals[name]
	if found {
		return fmt.Errorf("There is already an established thread local for %s", name)
	}

	operation := &threadLocalOperators{
		initializer: initializer,
		destroyer:   destroyer,
		lock:        goth.NewGoetheLock(),
		actuals:     make(map[int64]goethe.ThreadLocal),
	}

	goth.locals.threadLocals[name] = operation

	return nil
}

// Get thread local returns the instance of the storage associated with
// the current goethe thread.  May only be called on goethe threads and
// will return ErrNotGoetheThread if called from a non-goethe thread.
// If EstablishThreadLocal with the given name has not been called prior to
// this function call then ErrNoThreadLocalEstablished will be returned
func (goth *goetheData) GetThreadLocal(name string) (goethe.ThreadLocal, error) {
	tid := goth.GetThreadID()
	if tid < int64(0) {
		return nil, goethe.ErrNotGoetheThread
	}

	operators, found := goth.getOperatorsByName(name)
	if !found {
		operators = &threadLocalOperators{
			lock:    goth.NewGoetheLock(),
			actuals: make(map[int64]goethe.ThreadLocal),
		}

		goth.locals.localsMux.Lock()
		goth.locals.threadLocals[name] = operators
		goth.locals.localsMux.Unlock()
	}

	operators.lock.WriteLock()
	defer operators.lock.WriteUnlock()

	actual, found := operators.actuals[tid]
	if !found {
		actual = internal.NewThreadLocal(name, goth, tid)

		if operators.initializer != nil {
			operators.initializer(actual)
		}

		operators.actuals[tid] = actual
	}

	return actual, nil
}

func (goth *goetheData) startTimer() {
	goth.timers.timerMux.Lock()
	defer goth.timers.timerMux.Unlock()

	if goth.timers.timer != nil {
		return
	}

	goth.timers.timer = newTimer()

	// Add system job
	values := make([]reflect.Value, 0)
	goth.timers.timer.addJob(0, 24*time.Hour, nil,
		func() {
		}, values, false)

	goth.Go(goth.timers.timer.run)

	goth.EstablishThreadLocal(goethe.TimerThreadLocal, nil, nil)
}

// ScheduleAtFixedRate schedules the given method with the given args at
// a fixed rate.  The duration of the method does not affect when the
// next method will be run.  The first run will happen only after initialDelay
// and will then be scheduled at multiples of the period.  An optional
// error queue can be given to collect all errors thrown from the method.
// It is the responsibility of the caller to drain the error queue
func (goth *goetheData) ScheduleAtFixedRate(initialDelay time.Duration, period time.Duration,
	errorQueue goethe.ErrorQueue, method interface{}, args ...interface{}) (goethe.Timer, error) {
	goth.startTimer()

	argArray := make([]interface{}, len(args))
	for index, arg := range args {
		argArray[index] = arg
	}

	arguments, err := GetValues(method, argArray)
	if err != nil {
		return nil, err
	}

	return goth.timers.timer.addJob(initialDelay, period, errorQueue, method, arguments, true)
}

// ScheduleWithFixedDelay schedules the given method with the given args
// and will schedule the next run after the method returns and the delay has passed.
// The first run will happen only after initialDelay
// An optional error queue can be given to collect all errors thrown from the method.
// It is the responsibility of the caller to drain the error queue
func (goth *goetheData) ScheduleWithFixedDelay(initialDelay time.Duration, delay time.Duration,
	errorQueue goethe.ErrorQueue, method interface{}, args ...interface{}) (goethe.Timer, error) {
	goth.startTimer()

	argArray := make([]interface{}, len(args))
	for index, arg := range args {
		argArray[index] = arg
	}

	arguments, err := GetValues(method, argArray)
	if err != nil {
		return nil, err
	}

	return goth.timers.timer.addJob(initialDelay, delay, errorQueue, method, arguments, false)
}

func (goth *goetheData) getOperatorsByName(name string) (*threadLocalOperators, bool) {
	goth.locals.localsMux.Lock()
	goth.locals.localsMux.Unlock()

	retVal, found := goth.locals.threadLocals[name]

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
	goth.locals.localsMux.Lock()
	goth.locals.localsMux.Unlock()

	for _, operators := range goth.locals.threadLocals {
		removeThreadLocal(operators, tid)
	}
}

func (goth *goetheData) removePool(name string) {
	goth.pools.poolMux.Lock()
	goth.pools.poolMux.Unlock()

	delete(goth.pools.poolMap, name)
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

	Invoke(userCall, args, nil)

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
