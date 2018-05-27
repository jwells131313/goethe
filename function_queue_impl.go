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

package goethe

import (
	"sync"
	"time"
)

// FunctionQueueImpl is an implementation of a function queue
// that implements the FunctionQueue interface.  It has a
// bounded capacity and can take a callback function that
// is invoked any time an operation happens that changes
// the size of the queue
type FunctionQueueImpl struct {
	mux     sync.Mutex
	cond    *sync.Cond
	changer func(queue FunctionQueue)

	capacity uint32
	queue    []*FunctionDescriptor
}

// NewFunctionQueue creates a new function queue with the given capacity
func NewFunctionQueue(userCapacity uint32) FunctionQueue {
	retVal := &FunctionQueueImpl{
		capacity: userCapacity,
		queue:    make([]*FunctionDescriptor, 0),
	}

	retVal.cond = sync.NewCond(&retVal.mux)

	return retVal
}

// Enqueue queues a function to be run in the pool.  Returns
// ErrAtCapacity if the queue is currently at capacity
func (fq *FunctionQueueImpl) Enqueue(userCall interface{}, args ...interface{}) error {
	if userCall == nil {
		return nil
	}

	fq.mux.Lock()
	defer fq.mux.Unlock()

	if uint32(len(fq.queue)) >= fq.capacity {
		return ErrAtCapacity
	}

	descriptor := &FunctionDescriptor{
		UserCall: userCall,
		Args:     make([]interface{}, len(args)),
	}

	for index, arg := range args {
		descriptor.Args[index] = arg
	}

	fq.queue = append(fq.queue, descriptor)

	fq.cond.Broadcast()
	if fq.changer != nil {
		go fq.changer(fq)
	}

	return nil
}

// Dequeue returns a function to be run, waiting the given
// duration.  If there is no message within the given
// duration return the error returned will be ErrEmptyQueue
func (fq *FunctionQueueImpl) Dequeue(duration time.Duration) (*FunctionDescriptor, error) {
	fq.mux.Lock()
	defer fq.mux.Unlock()

	currentTime := time.Now()
	elapsedDuration := time.Since(currentTime)

	for (duration > 0) && (elapsedDuration < duration) && (len(fq.queue) <= 0) {
		timer := time.AfterFunc(duration-elapsedDuration, func() {
			fq.cond.Broadcast()
		})

		fq.cond.Wait()

		timer.Stop()

		elapsedDuration = time.Since(currentTime)
	}

	if len(fq.queue) <= 0 {
		return nil, ErrEmptyQueue
	}

	retVal := fq.queue[0]
	fq.queue = fq.queue[1:]

	if fq.changer != nil {
		go fq.changer(fq)
	}

	return retVal, nil
}

// GetCapacity gets the capacity of this queue
func (fq *FunctionQueueImpl) GetCapacity() uint32 {
	return fq.capacity
}

// GetSize returns the number of items currently in the queue
func (fq *FunctionQueueImpl) GetSize() int {
	fq.mux.Lock()
	defer fq.mux.Unlock()

	return len(fq.queue)
}

// IsEmpty Returns true if this queue is currently empty
func (fq *FunctionQueueImpl) IsEmpty() bool {
	return fq.GetSize() <= 0
}

// SetStateChangeCallback sets a function to be
// called whenever an enqueue or dequeue changes
// the size of queue
func (fq *FunctionQueueImpl) SetStateChangeCallback(ch func(FunctionQueue)) {
	fq.mux.Lock()
	defer fq.mux.Unlock()

	fq.changer = ch
}
