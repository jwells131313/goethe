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

package timers

import (
	"fmt"
	"github.com/jwells131313/goethe"
	"github.com/jwells131313/goethe/queues"
	"reflect"
	"sync/atomic"
	"time"
)

// Job represents a job in the current heap.  A job can be cancelled before it runs.
// After it runs a job appears to have been cancelled
type Job interface {
	goethe.Canceller

	GetTimeToRun() time.Time
}

// TimerHeap will run methods or functions after the specified duration or at a
// specific time.  The advantage of the TimerHeap is that the timer is only set
// to the next job that is to be processed, thus reducing timer ticks
//
// NOTE: This API is currently experimental and may change in the future
type TimerHeap interface {
	goethe.Canceller

	// GetErrorChannel returns the error channel for jobs that fail
	GetErrorChannel() <-chan error

	// AddJobByDuration adds a job to this heap with the given duration.  If the
	// duration is zero or negative the job will be run immediately.  Takes a method
	// or function and the arguments of that function.  The function should not take
	// long to execute.  Any long-running work that needs to be accomplished should
	// be done in a separate thread
	AddJobByDuration(time.Duration, interface{}, ...interface{}) (Job, error)

	// AddJobByDuration adds a job to this heap at the given time.  If the
	// time is now or in the past the job will be run immediately.  Takes a method
	// or function and the arguments to that function.  The function should not take
	// long to execute.  Any long-running work that needs to be accomplished should
	// be done in a separate thread
	AddJobByTime(time.Time, interface{}, ...interface{}) (Job, error)
}

type jobData struct {
	lock       goethe.Lock
	id         uint64
	timeToFire *time.Time
	closed     bool
	parent     *timerHeapData
	method     interface{}
	args       []reflect.Value
}

type timerData struct {
	jobID    uint64
	nextRing *time.Time
	timer    *time.Timer
}

type timerHeapData struct {
	lock         goethe.Lock
	heap         queues.Heap
	errorChannel chan error
	closed       bool
	name         string
	idFactory    uint64
	tInfo        *timerData
}

// NewTimerHeap creates a TimerHeap which is simply a set of jobs to
// be run at some time, where the intent is to minimize the number of
// outstanding timers.  In general there will be one timer running per
// TimerHeap, although at certain times there can ephemerally be
// more than one.  If the input errorChannel is not nil then if the
// function given to the job's last return argument is a non-nil
// implementation of error it will be put onto that channel
//
// NOTE: This API is currently experimental and may change in the future
func NewTimerHeap(errorChannel chan error) TimerHeap {
	retVal := &timerHeapData{
		heap:         queues.NewHeap(jobComparator),
		errorChannel: errorChannel,
		lock:         goethe.GG().NewGoetheLock(),
	}

	return retVal
}

func (thd *timerHeapData) Cancel() {
	tid := goethe.GG().GetThreadID()
	if tid < 0 {
		ch := make(chan bool)
		goethe.GG().Go(func() {
			thd.channelCancel(ch)
		})
		<-ch
		return
	}

	thd.internalCancel()
}

func (thd *timerHeapData) channelCancel(ch chan bool) {
	thd.internalCancel()
	ch <- true
}

func (thd *timerHeapData) internalCancel() {
	thd.lock.WriteLock()
	defer thd.lock.WriteUnlock()

	if thd.tInfo != nil {
		thd.tInfo.timer.Stop()

		thd.tInfo = nil
	}

	thd.closed = true

	if thd.errorChannel != nil {
		close(thd.errorChannel)
	}

}

func (thd *timerHeapData) IsRunning() bool {
	tid := goethe.GG().GetThreadID()
	if tid < 0 {
		ch := make(chan bool)
		goethe.GG().Go(func() {
			thd.channelIsRunning(ch)
		})
		return <-ch
	}

	return thd.internalChannelIsRunning()
}

func (thd *timerHeapData) channelIsRunning(ch chan bool) {
	ch <- thd.internalChannelIsRunning()
}

func (thd *timerHeapData) internalChannelIsRunning() bool {
	thd.lock.ReadLock()
	defer thd.lock.ReadUnlock()

	return !thd.closed
}

func (thd *timerHeapData) GetErrorChannel() <-chan error {
	thd.lock.ReadLock()
	defer thd.lock.ReadUnlock()

	return thd.errorChannel
}

func (thd *timerHeapData) AddJobByDuration(d time.Duration, method interface{}, args ...interface{}) (Job, error) {
	return thd.AddJobByTime(time.Now().Add(d), method, args...)
}

func (thd *timerHeapData) AddJobByTime(timeToFire time.Time, method interface{}, args ...interface{}) (Job, error) {
	values, err := getValues(method, args...)
	if err != nil {
		return nil, err
	}

	id := atomic.AddUint64(&thd.idFactory, 1)

	ttf := timeToFire
	newJob := &jobData{
		lock:       goethe.GG().NewGoetheLock(),
		id:         id,
		parent:     thd,
		method:     method,
		args:       values,
		timeToFire: &ttf,
	}

	thd.heap.Add(newJob)

	goethe.GG().Go(thd.scheduleNextTimer)

	return newJob, nil
}

func (thd *timerHeapData) newTimerData(jobID uint64, now *time.Time, nextRing *time.Time) *timerData {
	duration := nextRing.Sub(*now)
	if duration <= 0 {
		return nil
	}

	return &timerData{
		jobID:    jobID,
		nextRing: nextRing,
		timer: time.AfterFunc(duration, func() {
			goethe.GG().Go(thd.doAllCurrentJobs)
		}),
	}
}

func (thd *timerHeapData) scheduleNextTimer() {
	thd.lock.WriteLock()
	defer thd.lock.WriteUnlock()

	rawJob, found := thd.heap.Peek()
	if !found {
		// Nothing to do, do nothing
		return
	}

	job := rawJob.(*jobData)
	if thd.tInfo == nil {
		timeToFire := job.timeToFire
		now := time.Now()

		thd.tInfo = thd.newTimerData(job.id, &now, timeToFire)

		if thd.tInfo == nil {
			goethe.GG().Go(thd.doAllCurrentJobs)
		}

		return
	}

	if job.id == thd.tInfo.jobID {
		// This job is already the one whose timer is going to ring
		return
	}

	// Need to switch the timer.  First, cancel the old one
	thd.tInfo.timer.Stop()
	thd.tInfo = nil

	timeToFire := job.timeToFire
	now := time.Now()

	thd.tInfo = thd.newTimerData(job.id, &now, timeToFire)

	if thd.tInfo == nil {
		goethe.GG().Go(thd.doAllCurrentJobs)
	}
}

func (jd *jobData) Cancel() {
	tid := goethe.GG().GetThreadID()
	if tid < 0 {
		bc := make(chan bool)
		goethe.GG().Go(func() {
			jd.channelCancel(bc)
		})
		<-bc

		return
	}

	jd.internalCancel()
}

func (jd *jobData) channelCancel(bc chan bool) {
	jd.internalCancel()
	bc <- true
}

func (jd *jobData) internalCancel() {
	jd.lock.WriteLock()
	defer jd.lock.WriteUnlock()

	jd.closed = true
}

func (jd *jobData) IsRunning() bool {
	tid := goethe.GG().GetThreadID()
	if tid < 0 {
		bc := make(chan bool)
		goethe.GG().Go(func() {
			jd.channelIsRunning(bc)
		})
		return <-bc
	}

	return jd.internalIsRunning()
}

func (jd *jobData) channelIsRunning(bc chan bool) {
	bc <- jd.internalIsRunning()
}

func (jd *jobData) internalIsRunning() bool {
	jd.lock.ReadLock()
	defer jd.lock.ReadUnlock()

	return !jd.closed
}

func (jd *jobData) GetTimeToRun() time.Time {
	tid := goethe.GG().GetThreadID()
	if tid < 0 {
		tc := make(chan time.Time)
		goethe.GG().Go(func() {
			jd.channelGetTimeToRun(tc)
		})
		return <-tc
	}

	return jd.internalGetTimeToRun()
}

func (jd *jobData) channelGetTimeToRun(tc chan time.Time) {
	tc <- jd.internalGetTimeToRun()
}

func (jd *jobData) internalGetTimeToRun() time.Time {
	jd.lock.ReadLock()
	defer jd.lock.ReadUnlock()

	return *jd.timeToFire
}

func (jd *jobData) String() string {
	return fmt.Sprintf("job id=%d ttf=%s", jd.id, jd.timeToFire.String())
}

func (thd *timerHeapData) doAllCurrentJobs() {
	thd.lock.WriteLock()
	defer thd.lock.WriteUnlock()

	if thd.closed {
		return
	}

	defer thd.scheduleNextTimer()

	for {
		jobRaw, found := thd.heap.Peek()
		if !found {
			return
		}

		job := jobRaw.(*jobData)
		if !job.IsRunning() {
			// Clear it out and go on to next one
			thd.heap.Get()

			continue
		}

		now := time.Now()
		timeToFire := job.timeToFire

		if now.Before(*timeToFire) {
			return
		}

		// Clear out the job we are about to run, it's day is done
		thd.heap.Get()

		// Not cancelled, is now or in the past.  Run it
		invoke(job.method, job.args, job.parent.errorChannel)

		job.Cancel()
	}
}

func jobComparator(a interface{}, b interface{}) int {
	aJob := a.(*jobData)
	bJob := b.(*jobData)

	if aJob.timeToFire.After(*bJob.timeToFire) {
		return -1
	}
	if aJob.timeToFire.Before(*bJob.timeToFire) {
		return 1
	}

	return 0
}

// getValues returns the reflection values for the arguments as specified by
// the method parameters.  Will fail if the wrong number of arguments is passed
// in or the arguments are not the correct type.  Otherwise will return
// the value versions of the arguments
func getValues(method interface{}, args ...interface{}) ([]reflect.Value, error) {
	typ := reflect.TypeOf(method)
	kin := typ.Kind()
	if kin != reflect.Func {
		return nil, fmt.Errorf("first argument of GetValues must be a function, it is %s", kin.String())
	}

	numIn := typ.NumIn()
	if numIn != len(args) {
		return nil, fmt.Errorf("Method has %d parameters, user passed in %d", numIn, len(args))
	}

	expectedTypes := make([]reflect.Type, numIn)
	for i := 0; i < numIn; i++ {
		expectedTypes[i] = typ.In(i)
	}

	arguments := make([]reflect.Value, numIn)
	for index, arg := range args {
		var argValue reflect.Value
		if arg == nil {
			argValue = reflect.New(expectedTypes[index]).Elem()
		} else {
			argValue = reflect.ValueOf(arg)
		}

		arguments[index] = argValue

		if !argValue.Type().AssignableTo(expectedTypes[index]) {
			return nil, fmt.Errorf("Value at index %d of type %s does not match method parameter of type %s",
				index, argValue.Type().String(), expectedTypes[index].String())

		}
	}

	return arguments, nil
}

var (
	errorInterface = reflect.TypeOf((*error)(nil)).Elem()
)

// invoke will call the method with the arguments, and ship any errors
// returned by the method to the errorQueue (which may be nil)
func invoke(method interface{}, args []reflect.Value, echan chan error) {
	val := reflect.ValueOf(method)
	retVals := val.Call(args)

	if echan != nil && retVals != nil && len(retVals) > 0 {
		lastVal := retVals[len(retVals)-1]

		if !lastVal.IsNil() && lastVal.CanInterface() {
			// If last thing is an error and is not nil
			it := lastVal.Type()
			isAnError := it.Implements(errorInterface)

			if isAnError {
				iFace := lastVal.Interface()

				asErr := iFace.(error)

				echan <- asErr
			}
		}
	}
}
