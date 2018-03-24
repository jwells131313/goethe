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

// Package goethe is a package of thread utilities.  Threads created with
// goethe.Go have thread-ids unlike normal go routines.  Also within
// goethe threads you can use counting (recursive) read/write locks
// which are very useful when you are providing interface implementation
// to other users who may be using your api in threaded environments
package goethe

import "errors"

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

var (
	// ErrReadLockHeld returned if a WriteLock call is made while holding a ReadLock
	ErrReadLockHeld = errors.New("attempted to acquire a WriteLock while ReadLock was held")

	// ErrNotGoetheThread returned if any lock is attempted while not in a goethe thread
	ErrNotGoetheThread = errors.New("function called from non-goth thread")

	// ErrWriteLockNotHeld returned if a call to WriteUnlock is made while not holding the WriteLock
	ErrWriteLockNotHeld = errors.New("write lock is not held by this thread")
)
