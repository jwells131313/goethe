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

package tests

import (
	"github.com/jwells131313/goethe"
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type simpleValue struct {
	mux sync.Mutex

	value      int
	numReaders int32
}

type throttler struct {
	mux  sync.Mutex
	cond *sync.Cond

	proceed bool
}

func TestTwoWritersMutex(t *testing.T) {
	waiter := newSimpleValue()
	throttle := newThrottler()

	ethe := goethe.GetGoethe()
	lock := ethe.NewGoetheLock()

	ethe.Go(func() {
		var actualDepth int

		incrementValueByOne(lock, waiter, throttle, 0, &actualDepth)
	})

	ethe.Go(func() {
		var actualDepth int

		incrementValueByOne(lock, waiter, throttle, 0, &actualDepth)
	})

	received, gotValue := waiter.waitForValue(5, 1)
	if gotValue != true {
		t.Error("should have gotten to 1 very quickly, got ", received)
	}

	// Only ONE of the threads should get this, so after waiting
	// the value should only be one
	received, gotValue = waiter.waitForValue(2, 2)
	if gotValue {
		t.Error("should not have gotten the value 2", received)
		return
	}

	// Now, let the other thread go
	throttle.release()

	received, gotValue = waiter.waitForValue(5, 2)
	if !gotValue {
		t.Error("should have gotten the value 2", received)
		return
	}

	throttle.release()
}

func TestWriterWaitsForOneReader(t *testing.T) {
	writerWaitsForNReaders(t, 1, 0, 0)
}

func TestWriterWaitsForTenReaders(t *testing.T) {
	writerWaitsForNReaders(t, 10, 0, 0)
}

func TestWriterWaitsForOneCountingReader(t *testing.T) {
	writerWaitsForNReaders(t, 1, 5, 0)
}

func TestWriterWaitsForManyCountingReader(t *testing.T) {
	writerWaitsForNReaders(t, 5, 5, 0)
}

func TestCountingWriterWaitsForOneReader(t *testing.T) {
	writerWaitsForNReaders(t, 1, 0, 4)
}

func TestWriterCanBecomeReader(t *testing.T) {
	ethe := goethe.GetGoethe()
	lock := ethe.NewGoetheLock()
	gotHere := false

	ethe.Go(func() {
		lock.WriteLock()
		defer lock.WriteUnlock()

		lock.ReadLock()
		defer lock.ReadUnlock()

		gotHere = true
	})

	for lcv := 0; lcv < 200; lcv++ {
		if gotHere {
			// success
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Error("gotHere was not changed to true after 20 seconds")
}

func TestReaderCanNotBecomeWriter(t *testing.T) {
	ethe := goethe.GetGoethe()
	lock := ethe.NewGoetheLock()

	var err error

	ethe.Go(func() {
		lock.ReadLock()
		defer lock.ReadUnlock()

		err = lock.WriteLock()
	})

	for lcv := 0; lcv < 200; lcv++ {
		if err != nil {
			if err == goethe.ErrReadLockHeld {
				// success
				return
			}

			t.Errorf("unexpected error %v", err)
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Error("there was no error after 20 seconds")
}

func TestIsReadLockedInGThread(t *testing.T) {
	ethe := goethe.GetGoethe()
	lock := ethe.NewGoetheLock()

	retChan := make(chan bool)

	if !checkLocks(t, false, false, lock) {
		return
	}

	ethe.Go(func() {
		if !checkLocks(t, false, false, lock) {
			return
		}

		lock.ReadLock()

		if !checkLocks(t, true, false, lock) {
			return
		}

		lock.ReadUnlock()

		if !checkLocks(t, false, false, lock) {
			return
		}

		retChan <- true
	})

	ret := <-retChan
	assert.True(t, ret)

	if !checkLocks(t, false, false, lock) {
		return
	}
}

func TestIsWriteLockedInGThread(t *testing.T) {
	ethe := goethe.GetGoethe()
	lock := ethe.NewGoetheLock()

	retChan := make(chan bool)

	if !checkLocks(t, false, false, lock) {
		return
	}

	ethe.Go(func() {
		if !checkLocks(t, false, false, lock) {
			return
		}

		lock.WriteLock()

		if !checkLocks(t, false, true, lock) {
			return
		}

		lock.WriteUnlock()

		if !checkLocks(t, false, false, lock) {
			return
		}

		retChan <- true
	})

	ret := <-retChan
	assert.True(t, ret)

	if !checkLocks(t, false, false, lock) {
		return
	}
}

func TestIsLockedFromNonGThread(t *testing.T) {
	ethe := goethe.GetGoethe()
	lock := ethe.NewGoetheLock()

	syncer := make(chan int)

	// This thread generally does some lock operation then waits for the main
	// thread to test it.  The point is to test that the IsLocked verbs work
	// from non-goethe threads and return the correct results
	ethe.Go(func() {
		lock.ReadLock()

		syncer <- 0
		val := <-syncer
		assert.Equal(t, 0, val, "First sync bad")

		lock.ReadUnlock()

		syncer <- 1
		val = <-syncer
		assert.Equal(t, 1, val, "Second sync bad")

		lock.WriteLock()

		syncer <- 2
		val = <-syncer
		assert.Equal(t, 2, val, "Third sync bad")

		lock.WriteUnlock()

		syncer <- 3
		val = <-syncer
		assert.Equal(t, 3, val, "Fourth sync bad")

		lock.WriteLock()
		lock.ReadLock()

		syncer <- 4
		val = <-syncer
		assert.Equal(t, 4, val, "Fifth sync bad")

		lock.ReadUnlock()
		lock.WriteUnlock()

		syncer <- 5
	})

	// Wait for read lock
	sync := <-syncer
	if !assert.Equal(t, 0, sync, "First sync fail") {
		return
	}

	if !checkLocks(t, true, false, lock) {
		return
	}

	// Tell thread to go
	syncer <- sync

	// Wait for read unlock
	sync = <-syncer
	if !assert.Equal(t, 1, sync, "Second sync fail") {
		return
	}

	if !checkLocks(t, false, false, lock) {
		return
	}

	// Tell thread to go
	syncer <- sync

	// Wait for write lock
	sync = <-syncer
	if !assert.Equal(t, 2, sync, "Third sync fail") {
		return
	}

	if !checkLocks(t, false, true, lock) {
		return
	}

	// Tell thread to go
	syncer <- sync

	// Wait for write unlock
	sync = <-syncer
	if !assert.Equal(t, 3, sync, "Fourth sync fail") {
		return
	}

	if !checkLocks(t, false, false, lock) {
		return
	}

	// Tell the thread to go
	syncer <- sync

	// Wait for write/read lock
	sync = <-syncer
	if !assert.Equal(t, 4, sync, "Fifth sync fail") {
		return
	}

	if !checkLocks(t, true, true, lock) {
		return
	}

	// Tell the thread to go
	syncer <- sync

	// Wait for write/read unlock
	sync = <-syncer
	if !assert.Equal(t, 5, sync, "Sixth sync fail") {
		return
	}

	if !checkLocks(t, false, false, lock) {
		return
	}
}

/* ***************************************** Below find utility functions ****************************************** */
func checkLocks(t *testing.T, read, write bool, lock goethe.Lock) bool {
	r1 := assert.Equal(t, read, lock.IsReadLocked(), "IsReadLocked did not have correct reply")
	r2 := assert.Equal(t, write, lock.IsWriteLocked(), "IsWriteLocked did not have correct reply")

	bothExpected := read || write
	r3 := assert.Equal(t, bothExpected, lock.IsLocked(), "IsLocked did not have correct reply")

	return r1 && r2 && r3
}

func writerWaitsForNReaders(t *testing.T, numReaders int, recurseDepth int, writeRecurseDepth int) {
	waiter := newSimpleValue()
	throttle := newThrottler()

	ethe := goethe.GetGoethe()
	lock := ethe.NewGoetheLock()

	for lcv := 0; lcv < numReaders; lcv++ {
		ethe.Go(func() {
			var actualDepth int

			readValue(lock, waiter, throttle, recurseDepth, &actualDepth)
		})
	}

	expectedReaders := numReaders * (recurseDepth + 1)
	numReaders, foundReader := waiter.waitForNumReaders(10, expectedReaders)
	if !foundReader {
		t.Errorf("Did not get expected number of readers (%d) in 5 seconds, got %d",
			expectedReaders, numReaders)
		return
	}

	// A reader is in there, now fire up the writer
	ethe.Go(func() {
		var actualDepth int

		incrementValueByOne(lock, waiter, throttle, writeRecurseDepth, &actualDepth)
	})

	// Writer should not get this as reader is still in there

	received, gotValue := waiter.waitForValue(2, 1)
	if gotValue {
		t.Error("should not have gotten to the value 1", received)
		return
	}

	// Now, let the reader thread go
	throttle.release()

	expectedWriteValue := writeRecurseDepth + 1
	received, gotValue = waiter.waitForValue(5, expectedWriteValue)
	if !gotValue {
		t.Errorf("should have gotten the value %d, instead got %d", expectedWriteValue, received)
		return
	}

	throttle.release()

}

func incrementValueByOne(lock goethe.Lock, waiter *simpleValue,
	throttle *throttler, recurseDepth int, actualDepth *int) {
	lock.WriteLock()
	defer lock.WriteUnlock()

	waiter.value++

	if recurseDepth > *actualDepth {
		*actualDepth = *actualDepth + 1

		incrementValueByOne(lock, waiter, throttle, recurseDepth, actualDepth)

		return
	}

	throttle.wait()
}

// readValue the point of it recursing is to test the countingness of the read locks
func readValue(lock goethe.Lock, waiter *simpleValue, throttle *throttler,
	recurseDepth int, actualDepth *int) int {
	lock.ReadLock()
	defer lock.ReadUnlock()

	atomic.AddInt32(&waiter.numReaders, 1)

	if recurseDepth > *actualDepth {
		*actualDepth = *actualDepth + 1

		retVal := readValue(lock, waiter, throttle, recurseDepth, actualDepth)

		atomic.AddInt32(&waiter.numReaders, -1)
		return retVal
	}

	if recurseDepth == *actualDepth {
		throttle.wait()
	}

	atomic.AddInt32(&waiter.numReaders, -1)

	return waiter.value
}

func newSimpleValue() *simpleValue {
	retVal := &simpleValue{}

	return retVal
}

func (waiter *simpleValue) waitForNumReaders(seconds, numReaders int) (int, bool) {
	iterations := seconds * 10

	for lcv := 0; lcv < iterations; lcv++ {
		if int(atomic.AddInt32(&waiter.numReaders, 0)) == numReaders {
			return numReaders, true
		}

		time.Sleep(100 * time.Millisecond)
	}

	retVal := int(atomic.AddInt32(&waiter.numReaders, 0))
	if retVal == numReaders {
		return numReaders, true
	}

	return retVal, false
}

func (waiter *simpleValue) waitForValue(seconds, expected int) (int, bool) {
	iterations := seconds * 10

	waiter.mux.Lock()

	for lcv := 0; lcv < iterations; lcv++ {
		if waiter.value == expected {
			waiter.mux.Unlock()
			return waiter.value, true
		}

		waiter.mux.Unlock()
		time.Sleep(100 * time.Millisecond)
		waiter.mux.Lock()
	}

	retVal := waiter.value
	waiter.mux.Unlock()

	if retVal == expected {
		return retVal, true
	}

	return retVal, false
}

func newThrottler() *throttler {
	retVal := &throttler{
		proceed: false,
	}

	retVal.cond = sync.NewCond(&retVal.mux)
	return retVal
}

func (throttle *throttler) release() {
	throttle.mux.Lock()
	defer throttle.mux.Unlock()

	throttle.proceed = true
	throttle.cond.Broadcast()
}

func (throttle *throttler) reset() {
	throttle.mux.Lock()
	defer throttle.mux.Unlock()

	throttle.proceed = false
}

func (throttle *throttler) wait() {
	throttle.mux.Lock()
	defer throttle.mux.Unlock()

	throttle.cond.Wait()
}
