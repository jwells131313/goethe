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

// Package goethe contains several useful threading utilities.  Threads created
// with Goethe.Go have thread-ids unlike normal go routines.  Also within
// goethe threads you can use counting (recursive) read/write locks
// which are helpful when you are providing interface implementation
// to other users who may be using your api in threaded environments
package goethe

import (
	"errors"
	"time"
)

// Goethe a service which runs your routines in threads
// that can have things such as threadIds and thread
// local storage
type Goethe interface {
	// Go Runs the given function with the given parameters
	// in a new go thread.  Will always allocate a new
	// thread-id
	Go(func() error)

	// GetthreadID Gets the current threadID.  Returns -1
	// if this is not a goethe thread.  Thread ids start at 10
	// as thread ids 0 through 9 are reserved for future use
	GetThreadID() int64

	// NewGoetheLock Creates a new goethe lock
	NewGoetheLock() Lock

	// NewBoundedFunctionQueue returns a function queue with the given capacity
	NewBoundedFunctionQueue(int32) FunctionQueue

	// NewErrorQueue returns an error queue with the given capacity.  If errors
	// are returned when the ErrorQueue is at capacity the new errors are dropped
	NewErrorQueue(uint32) ErrorQueue

	// NewPool creates a new thread pool with the given parameters.  The name is the
	// name of this pool and may not be empty.  It is an error to try to create more than
	// one open pool with the same name at the same time.
	// minThreads is the minimum number of  threads that this pool will maintain while it is open.
	// minThreads may be zero. maxThreads is the maximum number of threads this pool will ever
	// allocate simultaneously.  New threads will be allocated if all of the threads in the
	// pool are busy and the FunctionQueue is not empty (and the total number of threads is less
	// than maxThreads) maxThreads must be greater than or equal to minThreads.  Having min and max
	// threads both be zero is an error.  Having min and max threads be the same value implies
	// a fixed thread size pool.  The idleDecayDuration is how long the system will wait
	// while the number of threads is greater than minThreads before removing ending the
	// thread.  functionQueue may not be nil and is how functions are enqueued onto the
	// thread pool.  errorQueue may be nil but if not nil any error returned by the function
	// will be enqueued onto the errorQueue.  It is recommended that the implementation of
	// ErrorQueue have some sort of upper bound
	NewPool(name string, minThreads int32, maxThreads int32, idleDecayDuration time.Duration,
		functionQueue FunctionQueue, errorQueue ErrorQueue) (Pool, error)

	// GetPool returns a non-closed pool with the given name.  If not found second
	// value returned will be false
	GetPool(string) (Pool, bool)
}

// Pool is used to manage a thread pool.  Every thread pool has one
// function pool and zero or one error queue
type Pool interface {
	// IsStarted returns true if this queue has been started
	IsStarted() bool

	// Attempts to start this pool.  Returns an error if this pool has been closed
	Start() error

	// GetName Gets the name of this pool
	GetName() string

	// GetMinThreads the minimum number of threads for this pool
	GetMinThreads() int32

	// GetMaxThreads the maximum number of threads for this pool
	GetMaxThreads() int32

	// GetIdleDecayDuration returns the IdleDecayDuration of this
	// thread pool (the duration a thread must be idle before being
	// removed from the pool)
	GetIdleDecayDuration() time.Duration

	// GetCurrentThreadCount returns the current number of active threads
	// in this pool
	GetCurrentThreadCount() int32

	// GetFunctionQueue Returns the function queue associated with this pool
	GetFunctionQueue() FunctionQueue

	// GetErrorQueue returns the error queue associated with this pool
	GetErrorQueue() ErrorQueue

	// IsClosed returns true if this pool has been closed.  Will remove
	// this pool from Goethe's map of pools
	IsClosed() bool

	// Close closes this pool.  All work remaining will be completed, but
	// no new work will be accepted.  The system will stop reading from
	// the FunctionQueue, so any remaining jobs can be found on the function
	// queue
	Close()
}

// Lock is a reader/writer lock that is a counting lock
// There can be multiple readers at the same time but only
// one writer.  You CAN get a reader lock while inside a write
// lock.  No readers will be allowed in while a write-lock
// is waiting to get in.  If you just use the WriteLock calls
// this behaves like a counting mutex
type Lock interface {
	// ReadLock Locks for read.  Multiple readers on multiple threads
	// are allowed in simultaneously.  Is counting, but all locks must
	// be paired with ReadUnlock.  You may get a ReadLock while holding
	// a WriteLock.  May only be called from inside a Goethe thread
	ReadLock() error

	// ReadUnlock unlocks the read lock.  Will only truly leave
	// critical section as reader when count is zero
	ReadUnlock() error

	// WriteLock Locks for write.  Only one writer is allowed
	// into the critical section.  Once a WriteLock is requested
	// no more readers will be allowed into the critical section.
	// An ReadLockHeld error will be returned immediately if an attempt
	// is made to acquire a WriteLock when a ReadLock is held
	WriteLock() error

	// WriteUnlock unlocks write lock.  Will only truly leave
	// critical section as reader when count is zero
	WriteUnlock() error
}

// FunctionQueue a queue of functions to be enqueued and dequeued
// The system can use any FunctionQueue it is given or you can use
// the ones returned by Goethe.NewBoundedFunctionQueue
type FunctionQueue interface {
	// Enqueue queues a function to be run in the pool.  Returns
	// ErrAtCapacity if the queue is currently at capacity
	Enqueue(func() error) error

	// Dequeue returns a function to be run, waiting the given
	// duration.  If there is no message within the given
	// duration return the error returned will be ErrEmptyQueue
	Dequeue(time.Duration) (func() error, error)

	// GetCapacity gets the capacity of this queue
	GetCapacity() int32

	// GetSize returns the number of items currently in the queue
	GetSize() int32

	// IsEmpty Returns true if this queue is currently empty
	IsEmpty() bool
}

// ErrorInformation represents data about an error that occurred
type ErrorInformation interface {
	// GetThreadID returns the thread id on which the error occurred
	GetThreadID() int64

	// GetError returns the error that occurred
	GetError() error
}

// ErrorQueue is used to retrieve errors thrown by the functions
// given to the thread pool.  Any implementation of this interface
// can be used by the system, or you can use the ones returned by
// Goethe.NewErrorQueue
type ErrorQueue interface {
	// Enqueue adds an error to the error queue.  If the queue is
	// at capacity should return ErrAtCapacity.  All other errors
	// will be ignored
	Enqueue(ErrorInformation) error

	// Dequeue removes ErrorInformation from the pools
	// error queue.  If there were no errors on the queue
	// the second return value is false
	Dequeue() (ErrorInformation, bool)
}

var (
	// ErrReadLockHeld returned if a WriteLock call is made while holding a ReadLock
	ErrReadLockHeld = errors.New("attempted to acquire a WriteLock while ReadLock was held")

	// ErrNotGoetheThread returned if any lock is attempted while not in a goethe thread
	ErrNotGoetheThread = errors.New("function called from non-goth thread")

	// ErrWriteLockNotHeld returned if a call to WriteUnlock is made while not holding the WriteLock
	ErrWriteLockNotHeld = errors.New("write lock is not held by this thread")

	// ErrAtCapacity returned by FunctionQueue.Enqueue if the queue is currently at capacity
	ErrAtCapacity = errors.New("queue is at capacity")

	// ErrEmptyQueue returned by FunctionQueue.Dequeue if no function was available inside
	// of the given duration
	ErrEmptyQueue = errors.New("queue is empty")
)
