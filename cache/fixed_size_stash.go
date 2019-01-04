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
// When an item is removed from the stash with Get or WaitForElement a new item
// will be added to the stash in the background with the create function.  Using a
// stash is useful when the creation time of an object is high but  many of them may be needed very quickly
// but sporadically.  The size of the stash should be determined by how high the expected peak request
// rate may become.
// For example, if it takes five minutes to create one of the items in the
// stash and its known that once an hour a burst of 15 requests come in at
// the same time it would be good to set the size to 15 to handle all of those
// requests quickly.  The system will run the StashCreateFunction in separate goethe
// threads, up to the maximum number of concurrent threads set on the stash
type FixedSizeStash interface {
	// GetDesiredSize Returns the desired size of the fixed size stash
	GetDesiredSize() int
	// GetCurrentSize Returns the current size of the fixed size stash (how many are currently available)
	GetCurrentSize() int
	// GetCreateFunction Returns the create function used to create elements of the stash
	GetCreateFunction() StashCreateFunction
	// GetErrorChannel returns the error channel for when the create function fails
	GetErrorChannel() <-chan error
	// GetMaximumConcurrency returns the maximum number of goroutines that will be used to create
	// new elements.  If this is zero then this stash will no longer attempt to create new elements
	GetMaximumConcurrency() int
	// SetMaximumConcurrency will set the maximum number of goroutines that will be used to create
	// new elements.  A value of zero will stop the stash from creating new elements.  A negative
	// number will cause a panic
	SetMaximumConcurrency(int)
	// Get returns an element from the stash, or false if there were no elements available
	Get() (interface{}, bool)
	// WaitForElement returns an element from the stash, and will block until an element becomes available
	// A duration of zero or less than zero will return immediately with an available element or an error
	WaitForElement(howLong time.Duration) (interface{}, error)
}

// StashElementDestructor An implementation of this interface will
// be passed to a stash element that implements ElementDestructorSetter
// It can be used to remove the element from the stash even if the element
// was never given to the user via the normal stash mechanism
type StashElementDestructor interface {
	// Removes the element from the stash.  Returns true
	// if the element was actually removed, or false if the
	// element was either removed via the normal stash methods
	// (Get or WaitForElement) or has already been destroyed
	DestroyElement() bool
}

// ElementDestructorSetter if an element created by the create function
// of the stash implements this interface then it will be called with
// a method that can be used to remove that element from the stash
type ElementDestructorSetter interface {
	// This method will be called immediately after an element
	// is created.  The destructor passed in can be used
	// to remove the element from the stash
	SetElementDestructor(StashElementDestructor)
}

type fixedSizeStashData struct {
	lock               goethe.Lock
	cond               *sync.Cond
	creator            StashCreateFunction
	desiredSize        int
	maxConcurrency     int
	errorChannel       chan error
	elements           *dll
	outstandingCreates int
	tHeap              timers.TimerHeap
	numWorkers         int
}

// NewFixedSizeStash returns a FixedSizeStash and will immediately start creating items in the stash
// The desired size must be at least two, and creator must not be nil.  eChan may be nil
func NewFixedSizeStash(creator StashCreateFunction, desiredSize int, maxConcurrency int, eChan chan error) FixedSizeStash {
	if creator == nil {
		panic("creator may not be nil")
	}
	if desiredSize < 2 {
		ec := fmt.Sprintf("Requested size of stash (%d) must be at least two", desiredSize)
		panic(ec)
	}
	if maxConcurrency < 0 {
		ec := fmt.Sprintf("Requested maximum concurrency of stash (%d) must be greater than or equal to 0", maxConcurrency)
		panic(ec)
	}

	retVal := &fixedSizeStashData{
		lock:           gd.NewGoetheLock(),
		creator:        creator,
		desiredSize:    desiredSize,
		maxConcurrency: maxConcurrency,
		errorChannel:   eChan,
		tHeap:          timers.NewTimerHeap(eChan),
	}
	retVal.elements = newDLL(retVal)

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

func (fssd *fixedSizeStashData) GetMaximumConcurrency() int {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan int)

		gd.Go(fssd.channelGetMaxConcurrency, replyChan)

		return <-replyChan
	}

	return fssd.internalGetMaxConcurrency()
}

func (fssd *fixedSizeStashData) channelGetMaxConcurrency(ret chan int) {
	ret <- fssd.internalGetMaxConcurrency()
}

func (fssd *fixedSizeStashData) internalGetMaxConcurrency() int {
	fssd.lock.ReadLock()
	defer fssd.lock.ReadUnlock()

	return fssd.maxConcurrency
}

func (fssd *fixedSizeStashData) SetMaximumConcurrency(max int) {
	if max < 0 {
		ec := fmt.Sprintf("Requested maximum concurrency of stash (%d) must be greater than or equal to 0", max)
		panic(ec)
	}

	tid := gd.GetThreadID()
	if tid < 0 {
		dChan := make(chan bool)

		gd.Go(fssd.channelSetMaximumConcurrency, max, dChan)

		<-dChan

		return
	}

	fssd.internalSetMaximumConcurrency(max)
}

func (fssd *fixedSizeStashData) channelSetMaximumConcurrency(max int, dChan chan bool) {
	fssd.internalSetMaximumConcurrency(max)
	dChan <- true
}

func (fssd *fixedSizeStashData) internalSetMaximumConcurrency(max int) {
	fssd.lock.WriteLock()
	defer fssd.lock.WriteUnlock()

	oldValue := fssd.maxConcurrency
	fssd.maxConcurrency = max

	if oldValue < fssd.maxConcurrency {
		gd.Go(fssd.builder)
	}
}

func (fssd *fixedSizeStashData) channelInternalGetSize(ret chan int) {
	ret <- fssd.internalGetSize()
}

func (fssd *fixedSizeStashData) internalGetSize() int {
	fssd.lock.ReadLock()
	defer fssd.lock.ReadUnlock()

	return fssd.elements.size
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
		size := fssd.elements.size
		if size > 0 {
			e0, _ := fssd.elements.Remove()

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
	fssd.lock.WriteLock()
	defer fssd.lock.WriteUnlock()

	for {
		if fssd.numWorkers >= fssd.maxConcurrency {
			return
		}

		size := fssd.elements.size
		total := size + fssd.numWorkers

		if total >= fssd.desiredSize {
			return
		}

		fssd.numWorkers++
		gd.Go(fssd.createElements)
	}
}

func (fssd *fixedSizeStashData) createElements() {
	for {
		if !fssd.doGoOn() {
			return
		}

		fssd.expand()
	}
}

func (fssd *fixedSizeStashData) expand() {
	elem, err := fssd.createOne()
	if err != nil {
		fssd.lock.WriteLock()
		defer fssd.lock.WriteUnlock()

		fssd.outstandingCreates--
		if fssd.outstandingCreates < 0 {
			fssd.outstandingCreates = 0
		}

		return
	}

	fssd.lock.WriteLock()
	defer fssd.lock.WriteUnlock()

	fssd.outstandingCreates--
	if fssd.outstandingCreates < 0 {
		fssd.outstandingCreates = 0
	}

	destructor := fssd.elements.Add(elem)
	asInjectee, ok := elem.(ElementDestructorSetter)
	if ok {
		asInjectee.SetElementDestructor(destructor)
	}

	fssd.cond.Broadcast()
}

func (fssd *fixedSizeStashData) doGoOn() bool {
	fssd.lock.WriteLock()
	defer fssd.lock.WriteUnlock()

	size := fssd.elements.size
	totalPotential := size + fssd.outstandingCreates

	if totalPotential >= fssd.desiredSize {
		fssd.numWorkers--
		if fssd.numWorkers < 0 {
			fssd.numWorkers = 0
		}

		return false
	}

	fssd.outstandingCreates = fssd.outstandingCreates + 1
	return true
}

func (fssd *fixedSizeStashData) createOne() (ret interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			ret = nil

			err = fmt.Errorf("%v", r)

			if fssd.errorChannel != nil {
				fssd.errorChannel <- err
			}
		}
	}()

	ret, err = fssd.creator()
	if err != nil {
		if fssd.errorChannel != nil {
			fssd.errorChannel <- err
		}

		return
	}
	if ret == nil {
		err = fmt.Errorf("The creator function for a stash returned a nil element and nil error")
		fssd.errorChannel <- err

		return
	}

	return
}

type dllNode struct {
	payload interface{}
	deleted bool
	parent  *dll
	next    *dllNode
	prev    *dllNode
}

// dll this implementation of dll is protected by outside locks
type dll struct {
	head   *dllNode
	tail   *dllNode
	size   int
	parent *fixedSizeStashData
}

func newDllNode(element interface{}, parent *dll) *dllNode {
	return &dllNode{
		payload: element,
		parent:  parent,
	}
}

func newDLL(parent *fixedSizeStashData) *dll {
	return &dll{
		parent: parent,
	}
}

func (dll *dll) Add(element interface{}) StashElementDestructor {
	newNode := newDllNode(element, dll)

	dll.size = dll.size + 1

	currentHead := dll.head
	if currentHead == nil {
		dll.head = newNode
		dll.tail = newNode
		return newNode
	}

	newNode.next = currentHead
	currentHead.prev = newNode

	dll.head = newNode

	return newNode
}

func (dll *dll) Remove() (interface{}, bool) {
	currentTail := dll.tail
	if currentTail == nil {
		return nil, false
	}

	dll.size = dll.size - 1

	currentTail.deleted = true

	previousNode := currentTail.prev
	currentTail.prev = nil

	if previousNode == nil {
		// Last one
		dll.head = nil
		dll.tail = nil

		return currentTail.payload, true
	}

	dll.tail = previousNode

	previousNode.next = nil

	return currentTail.payload, true
}

func (node *dllNode) DestroyElement() bool {
	sd := node.parent.parent
	if sd == nil {
		// no lock needed
		return node.internalDestroyElement()
	}

	tid := gd.GetThreadID()
	if tid < 0 {
		ch := make(chan bool)

		gd.Go(node.channelDestroyElement, ch)

		return <-ch
	}

	sd.lock.WriteLock()
	defer sd.lock.WriteUnlock()

	return node.internalDestroyElement()
}

func (node *dllNode) channelDestroyElement(ch chan bool) {
	sd := node.parent.parent

	sd.lock.WriteLock()
	defer sd.lock.WriteUnlock()

	retVal := node.internalDestroyElement()
	ch <- retVal
}

func (node *dllNode) internalDestroyElement() bool {
	if node.deleted {
		return false
	}

	dll := node.parent
	dll.size = dll.size - 1
	if dll.parent != nil {
		gd.Go(dll.parent.builder)
	}

	previousNode := node.prev
	nextNode := node.next

	node.deleted = true
	node.next = nil
	node.prev = nil

	if previousNode == nil {
		dll.head = nextNode
	} else {
		previousNode.next = nextNode
	}

	if nextNode == nil {
		dll.tail = previousNode
	} else {
		nextNode.prev = previousNode
	}

	return true
}
