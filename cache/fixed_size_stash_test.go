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

package cache

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"
)

func TestConstantStashOutput(t *testing.T) {
	eChan := make(chan error)
	f := func() (interface{}, error) {
		return "", nil
	}

	stash := NewFixedSizeStash(f, 10, 5, eChan)
	if !assert.NotNil(t, stash) {
		return
	}

	g := stash.GetCreateFunction()
	assert.NotNil(t, g)

	ret, err := g()
	assert.Nil(t, err)
	assert.Equal(t, "", ret)

	assert.Equal(t, 10, stash.GetDesiredSize())
	assert.NotNil(t, stash.GetErrorChannel())

	assert.Equal(t, 5, stash.GetMaximumConcurrency())

	stash.SetMaximumConcurrency(13)

	assert.Equal(t, 13, stash.GetMaximumConcurrency())
}

func TestSizeGetsToTen(t *testing.T) {
	f := func() (interface{}, error) {
		return "", nil
	}

	stash := NewFixedSizeStash(f, 10, 5, nil)
	if !assert.NotNil(t, stash) {
		return
	}

	for lcv := 0; lcv < 2000; lcv++ {
		size := stash.GetCurrentSize()
		assert.True(t, size <= 10)

		if size >= 10 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, 10, stash.GetCurrentSize())

	time.Sleep(100 * time.Millisecond)

	// Like, make sure no more are created
	assert.Equal(t, 10, stash.GetCurrentSize())
}

// TestGetAFewElements gets a few elements, but does not go over the stash size
func TestGetAFewElements(t *testing.T) {
	f := func() (interface{}, error) {
		return "", nil
	}

	stash := NewFixedSizeStash(f, 10, 5, nil)
	if !assert.NotNil(t, stash) {
		return
	}

	// sleep to give time to get elements
	time.Sleep(100 * time.Millisecond)

	for lcv := 0; lcv < 8; lcv++ {
		elem, got := stash.Get()
		if !assert.True(t, got) {
			return
		}
		if !assert.NotNil(t, elem) {
			return
		}

		if !assert.Equal(t, "", elem) {
			return
		}
	}

	for lcv := 0; lcv < 2000; lcv++ {
		size := stash.GetCurrentSize()
		assert.True(t, size <= 10)

		if size >= 10 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, 10, stash.GetCurrentSize())
}

// TestGetAFewElements gets a few elements, but does not go over the stash size
func TestGetAFewWithWaitFor(t *testing.T) {
	f := func() (interface{}, error) {
		return "", nil
	}

	stash := NewFixedSizeStash(f, 10, 5, nil)
	if !assert.NotNil(t, stash) {
		return
	}

	// No need to sleep, we give it enough time in WaitFor

	for lcv := 0; lcv < 8; lcv++ {
		elem, err := stash.WaitForElement(100 * time.Millisecond)
		if !assert.Nil(t, err) {
			return
		}
		if !assert.NotNil(t, elem) {
			return
		}

		if !assert.Equal(t, "", elem) {
			return
		}
	}

	for lcv := 0; lcv < 2000; lcv++ {
		size := stash.GetCurrentSize()
		assert.True(t, size <= 10)

		if size >= 10 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, 10, stash.GetCurrentSize())
}

// TestGetAFewElements gets a few elements, but does not go over the stash size
func TestNoElementInTime(t *testing.T) {
	f := func() (interface{}, error) {
		time.Sleep(5 * time.Minute)
		return "", nil
	}

	stash := NewFixedSizeStash(f, 10, 5, nil)
	if !assert.NotNil(t, stash) {
		return
	}

	// There isn't enough time for this to have an element, make sure we get the correct error
	_, err := stash.WaitForElement(100 * time.Millisecond)
	if !assert.Equal(t, err, ErrNoElementAvailable) {
		return
	}
}

// TestCheckMaximumConcurrency makes sure we only ever create ten threads
func TestCheckMaximumConcurrency(t *testing.T) {
	tidMap := make(map[int64]int64)

	f := func() (interface{}, error) {
		tid := gd.GetThreadID()
		tidMap[tid] = tid

		sleepTime := rand.Intn(100) + 1
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)

		return "", nil
	}

	stash := NewFixedSizeStash(f, 100, 10, nil)
	if !assert.NotNil(t, stash) {
		return
	}

	for lcv := 0; lcv < 2000; lcv++ {
		size := stash.GetCurrentSize()
		assert.True(t, size <= 100)

		if size >= 100 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, 100, stash.GetCurrentSize())
	assert.Equal(t, 10, len(tidMap), "The map size should equal the max concurrency size (%v)", tidMap)
}

var discoveredErrors int32

// TestErrors puts some random errors in there
func TestErrors(t *testing.T) {
	tidMap := make(map[int64]int64)

	discoveredErrors = 0
	var numErrors int32

	f := func() (interface{}, error) {
		tid := gd.GetThreadID()
		tidMap[tid] = tid

		sleepTime := rand.Intn(100) + 1
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)

		if sleepTime%10 == 0 {
			val := atomic.AddInt32(&numErrors, 1)

			return nil, fmt.Errorf("There was an error on thread %d, number %d", tid, val)
		}

		return "", nil
	}

	eChan := make(chan error, 10)
	doneChan := make(chan bool)
	defer func() {
		doneChan <- true
	}()

	gd.Go(errorCounter, eChan, doneChan)

	stash := NewFixedSizeStash(f, 100, 10, eChan)
	if !assert.NotNil(t, stash) {
		return
	}

	for lcv := 0; lcv < 2000; lcv++ {
		size := stash.GetCurrentSize()
		assert.True(t, size <= 100)

		if size >= 100 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, 100, stash.GetCurrentSize())
	assert.Equal(t, 10, len(tidMap), "The map size should equal the max concurrency size (%v)", tidMap)

	// Sleep an extra half-second to catch straggling errors on the queue, or to see
	// if somehow some weird extra errors show up
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, numErrors, discoveredErrors)

}

// TestPanics puts some random panics in there
func TestPanics(t *testing.T) {
	tidMap := make(map[int64]int64)

	discoveredErrors = 0
	var numErrors int32

	f := func() (interface{}, error) {
		tid := gd.GetThreadID()
		tidMap[tid] = tid

		sleepTime := rand.Intn(100) + 1
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)

		if sleepTime%10 == 0 {
			val := atomic.AddInt32(&numErrors, 1)

			ec := fmt.Sprintf("There was an error on thread %d, number %d", tid, val)

			panic(ec)
		}

		return "", nil
	}

	eChan := make(chan error)
	doneChan := make(chan bool)
	defer func() {
		doneChan <- true
	}()

	gd.Go(errorCounter, eChan, doneChan)

	stash := NewFixedSizeStash(f, 100, 10, eChan)
	if !assert.NotNil(t, stash) {
		return
	}

	for lcv := 0; lcv < 2000; lcv++ {
		size := stash.GetCurrentSize()
		assert.True(t, size <= 100)

		if size >= 100 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, 100, stash.GetCurrentSize())
	assert.Equal(t, 10, len(tidMap), "The map size should equal the max concurrency size (%v)", tidMap)

	// Sleep an extra half-second to catch straggling errors on the queue, or to see
	// if somehow some weird extra errors show up
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, numErrors, discoveredErrors)

}

func errorCounter(eChan chan error, doneChan chan bool) {
	for {
		select {
		case <-eChan:
			atomic.AddInt32(&discoveredErrors, 1)
		case <-doneChan:
			return
		}
	}
}
