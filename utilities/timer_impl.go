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
	"github.com/jwells131313/goethe"
	"github.com/jwells131313/goethe/internal"
	"reflect"
	"sync"
	"time"
)

const (
	fudgeFactor time.Duration = 10 * time.Millisecond
)

type timerImpl interface {
	run()

	addJob(
		initialTime *time.Time,
		initialDelay time.Duration,
		period time.Duration,
		errorQueue goethe.ErrorQueue,
		method interface{},
		arguments []reflect.Value,
		fixed bool) (goethe.Timer, error)
}

type timerData struct {
	mux           goethe.Lock
	cond          *sync.Cond
	heap          internal.HeapQueue
	nextJobNumber int64
	sleepy        sleeper
	nextJob       uint64
}

type timerJob struct {
	mux         sync.Mutex
	jobNumber   uint64
	initialTime *time.Time
	cancelled   bool
	delay       time.Duration
	fixed       bool
	method      interface{}
	args        []reflect.Value
	errors      goethe.ErrorQueue
}

// NewTimer creates a timer for use with the goethe scheduler
func newTimer() timerImpl {
	goethe := GetGoethe()

	retVal := &timerData{
		mux:    goethe.NewGoetheLock(),
		heap:   internal.NewHeap(),
		sleepy: newSleeper(),
	}

	retVal.cond = sync.NewCond(retVal.mux)

	return retVal
}

func (timer *timerData) run() {
	for timer.runOne() {
	}
}

func (timer *timerData) runOne() bool {
	goethe := GetGoethe()

	timer.mux.Lock()
	defer timer.mux.Unlock()

	now := time.Now()
	now = now.Add(fudgeFactor)

	peek, peekNode, found := timer.heap.Peek()
	if !found {
		timer.cond.Wait()

		return true
	}

	pNode := peekNode.(*timerJob)

	// Is it inside the range
	until := (*peek).Sub(now)
	if until >= 0 {
		timer.sleepy.sleep(until, timer.cond, pNode.jobNumber)

		timer.cond.Wait()

		return true
	}

	pop, payload, found := timer.heap.Get()
	if !found {
		// should never happen because peek said it was there
		return true
	}

	job, ok := payload.(*timerJob)
	if !ok {
		// should never happen?
		return true
	}

	until = (*pop).Sub(now)
	if until >= 0 {
		// Also should never happen, peek said it was time
		timer.internalAddJob(nil, job.initialTime, 0, job.delay, job.errors, job.method, job.args, job.fixed)

		timer.sleepy.sleep(until, timer.cond, job.jobNumber)
		return true
	}

	// Ok, time to actually run the job!
	if !job.IsRunning() {
		// cancelled, just return
		return true
	}

	if job.fixed {
		// calculate the next time
		sinceInitial := now.Sub(*job.initialTime)

		numRuns := sinceInitial / job.delay

		nextIteration := numRuns + 1

		nextOffset := nextIteration * job.delay

		nextRunTime := job.initialTime.Add(nextOffset)

		timer.internalAddJob(nil, &nextRunTime, 0, job.delay, job.errors, job.method, job.args, job.fixed)
	}

	goethe.GoWithArgs(timer.invoke, job)

	peek, peekNode, found = timer.heap.Peek()
	if !found {
		return true
	}

	pNode = peekNode.(*timerJob)

	// Is it inside the range
	until = time.Until(*peek)
	timer.sleepy.sleep(until, timer.cond, pNode.jobNumber)

	return true
}

func (timer *timerData) invoke(job *timerJob) {
	Invoke(job.method, job.args, job.errors)

	if job.fixed {
		// parent put new job on
		return
	}

	nextRun := time.Now().Add(job.delay)

	timer.internalAddJob(nil, &nextRun, 0, job.delay, job.errors, job.method, job.args, job.fixed)
}

func (timer *timerData) addJob(
	initialTime *time.Time,
	initialDelay time.Duration,
	period time.Duration,
	errorQueue goethe.ErrorQueue,
	method interface{},
	arguments []reflect.Value,
	fixed bool) (goethe.Timer, error) {
	ethe := GetGoethe()

	retChannel := make(chan goethe.Timer)

	ethe.GoWithArgs(timer.internalAddJob,
		retChannel, initialTime, initialDelay, period, errorQueue, method, arguments, fixed)

	reply := <-retChannel

	return reply, nil
}

func (timer *timerData) internalAddJob(
	retChannel chan goethe.Timer,
	initialTime *time.Time,
	initialDelay time.Duration,
	period time.Duration,
	errorQueue goethe.ErrorQueue,
	method interface{},
	arguments []reflect.Value,
	fixed bool) (goethe.Timer, error) {

	var firstRun *time.Time
	if initialTime == nil {
		now := time.Now()
		added := now.Add(initialDelay)
		firstRun = &added
	} else {
		firstRun = initialTime
	}

	timer.mux.Lock()
	defer timer.mux.Unlock()

	nextJob := timer.nextJob
	timer.nextJob++

	job := &timerJob{
		initialTime: firstRun,
		delay:       period,
		fixed:       fixed,
		method:      method,
		args:        arguments,
		errors:      errorQueue,
		jobNumber:   nextJob,
	}

	err := timer.heap.Add(firstRun, job)
	if err != nil {
		return nil, err
	}

	timer.cond.Broadcast()

	if retChannel != nil {
		retChannel <- job
	}

	return job, nil
}

// Cancel cancels the timer
func (job *timerJob) Cancel() {
	job.mux.Lock()
	defer job.mux.Unlock()

	job.cancelled = true
}

// IsRunning true if this timer is running, false if it has been cancelled
func (job *timerJob) IsRunning() bool {
	job.mux.Lock()
	defer job.mux.Unlock()

	return !job.cancelled
}

// GetErrorQueue returns the error queue associated with this timer (may be nil)
func (job *timerJob) GetErrorQueue() goethe.ErrorQueue {
	return job.errors
}
