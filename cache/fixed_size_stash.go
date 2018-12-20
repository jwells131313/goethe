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
	"errors"
	"fmt"
	"github.com/jwells131313/goethe"
	"github.com/jwells131313/goethe/timers"
	"sync"
	"time"
)

// StashCreateFunction is the function that will be called, possibly in
// separate threads, to create an entity in the FixedSizeStash.  It is an
// error to return both a nil element and nil error
type StashCreateFunction func() (interface{}, error)

var (
	// ErrNoElementAvailable is returned from WaitForElement when there is no available element
	// within the given duration
	ErrNoElementAvailable = errors.New("There is no element currently available in the stash")
)

// FixedSizeStash is a stash of items that should be kept at a certain size
// When an item is removed from the stash with Get a new item will be added
// to the stash in the background with the create function.  Using a stash
// like this is useful when the creation time of an object is very high but
// many of them may be needed very quickly but sporadically.  The size of the
// stash should be determined by how high the expected peak request rate may become.
// For example, if it takes five minutes to create one of the items in the
// stash and its known that once an hour a burst of 15 requests come in at
// the same time it would be good to set the size to 15 to handle all of those
// requests quickly.  The system will run the StashCreateFunction
type FixedSizeStash interface {
	// GetDesiredSize Returns the desired size of the fixed size stash
	GetDesiredSize() int
	// GetCurrentSize Returns the current size of the fixed size stash (how many are currently available)
	GetCurrentSize() int
	// GetCreateFunction Returns the create function used to create elements of the stash
	GetCreateFunction() StashCreateFunction
	// GetErrorChannel returns the error channel for when the create function fails
	GetErrorChannel() <-chan error
	// Get returns an element from the stash, or false if there were no elements available
	Get() (interface{}, bool)
	// WaitForElement returns an element from the stash, and will block until an element becomes available
	// A duration of zero or less than zero will return immediately with an available element or an error
	WaitForElement(howLong time.Duration) (interface{}, error)
}

type fixedSizeStashData struct {
	lock               goethe.Lock
	cond               *sync.Cond
	creator            StashCreateFunction
	desiredSize        int
	errorChannel       chan error
	elements           []interface{}
	outstandingCreates int
	tHeap              timers.TimerHeap
}

// NewFixedSizeStash returns a FixedSizeStash and will immediately start creating items in the stash
// The desired size must be at least two, and creator must not be nil.  eChan may be nil
func NewFixedSizeStash(creator StashCreateFunction, desiredSize int, eChan chan error) FixedSizeStash {
	if creator == nil {
		panic("creator may not be nil")
	}
	if desiredSize < 2 {
		ec := fmt.Sprintf("Requested size of stash (%d) must be at least two", desiredSize)
		panic(ec)
	}

	retVal := &fixedSizeStashData{
		lock:         gd.NewGoetheLock(),
		creator:      creator,
		desiredSize:  desiredSize,
		errorChannel: eChan,
		elements:     make([]interface{}, 0),
		tHeap:        timers.NewTimerHeap(eChan),
	}

	retVal.cond = sync.NewCond(retVal.lock)

	gd.Go(retVal.builder)

	return retVal
}

func (fssd *fixedSizeStashData) GetDesiredSize() int {
	return fssd.desiredSize
}

func (fssd *fixedSizeStashData) GetCurrentSize() int {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan int)

		gd.Go(fssd.channelInternalGetSize, replyChan)

		return <-replyChan
	}

	return fssd.internalGetSize()
}

func (fssd *fixedSizeStashData) channelInternalGetSize(ret chan int) {
	ret <- fssd.internalGetSize()
}

func (fssd *fixedSizeStashData) internalGetSize() int {
	fssd.lock.ReadLock()
	defer fssd.lock.ReadUnlock()

	return len(fssd.elements)
}

func (fssd *fixedSizeStashData) GetCreateFunction() StashCreateFunction {
	return fssd.creator
}

func (fssd *fixedSizeStashData) GetErrorChannel() <-chan error {
	return fssd.errorChannel
}

func (fssd *fixedSizeStashData) Get() (interface{}, bool) {
	r, e := fssd.WaitForElement(0)
	if e != nil {
		return nil, false
	}
	return r, true
}

type waitFor struct {
	elem interface{}
	err  error
}

func (fssd *fixedSizeStashData) WaitForElement(howLong time.Duration) (interface{}, error) {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan *waitFor)

		gd.Go(fssd.channelWaitForElement, howLong, replyChan)

		wf := <-replyChan

		return wf.elem, wf.err
	}

	wf := fssd.internalWaitForElement(howLong)
	return wf.elem, wf.err
}

func (fssd *fixedSizeStashData) channelWaitForElement(howLong time.Duration, ret chan *waitFor) {
	ret <- fssd.internalWaitForElement(howLong)
}

func (fssd *fixedSizeStashData) internalWaitForElement(howLong time.Duration) *waitFor {
	fssd.lock.WriteLock()
	defer fssd.lock.WriteUnlock()

	var job timers.Job
	then := time.Now()
	for {
		size := len(fssd.elements)
		if size > 0 {
			e0 := fssd.elements[0]
			fssd.elements = fssd.elements[1:]

			if job != nil {
				job.Cancel()
			}

			gd.Go(fssd.builder)

			return &waitFor{
				elem: e0,
			}
		}

		elapsed := time.Now().Sub(then)
		leftToWait := howLong - elapsed

		if leftToWait <= 0 {
			return &waitFor{
				err: ErrNoElementAvailable,
			}
		}

		j, err := fssd.tHeap.AddJobByDuration(leftToWait, fssd.durationExpiry)
		if err != nil {
			return &waitFor{
				err: err,
			}
		}

		job = j
		fssd.cond.Wait()
	}
}

func (fssd *fixedSizeStashData) durationExpiry() {
	fssd.lock.WriteLock()
	defer fssd.lock.WriteUnlock()

	fssd.cond.Signal()
}

func (fssd *fixedSizeStashData) builder() {
	// This will call addOne probably more than it needs
	// to, but addOne will only actually try to add one
	// if there aren't already enough outstanding adds
	for lcv := 0; lcv < fssd.desiredSize; lcv++ {
		fssd.addOne()
	}
}

func (fssd *fixedSizeStashData) addOne() {
	fssd.lock.WriteLock()
	defer fssd.lock.WriteUnlock()

	size := len(fssd.elements)
	totalPotential := size + fssd.outstandingCreates

	if totalPotential >= fssd.desiredSize {
		return
	}

	fssd.outstandingCreates++
	gd.Go(fssd.createOne)
}

func (fssd *fixedSizeStashData) createOne() {
	// Do NOT lock while doing the create
	element, err := fssd.creator()
	if err != nil {
		if fssd.errorChannel != nil {
			fssd.errorChannel <- err
		}
		return
	}
	if element == nil {
		fssd.errorChannel <- fmt.Errorf("The creator function for a stash returned a nil element and nil error")
	}

	fssd.lock.WriteLock()
	defer fssd.lock.WriteUnlock()

	fssd.outstandingCreates--
	if fssd.outstandingCreates < 0 {
		fssd.outstandingCreates = 0
	}
	fssd.elements = append(fssd.elements, element)

	fssd.cond.Signal()
}
