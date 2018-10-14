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

type goetheLock struct {
	parent *StandardThreadUtilities

	goMux sync.Mutex
	cond  *sync.Cond

	readerCounts map[int64]int32

	holdingWriter  int64
	writerCount    int32
	writersWaiting int64
}

func newReaderWriterLock(pparent *StandardThreadUtilities) Lock {
	retVal := &goetheLock{
		parent:        pparent,
		holdingWriter: -2,
		readerCounts:  make(map[int64]int32),
	}

	retVal.cond = sync.NewCond(&retVal.goMux)

	return retVal
}

func (lock *goetheLock) Lock() {
	err := lock.WriteLock()
	if err != nil {
		panic(err)
	}
}

func (lock *goetheLock) Unlock() {
	err := lock.WriteUnlock()
	if err != nil {
		panic(err)
	}
}

// ReadLock Locks for read.  Multiple readers on multiple threads
// are allowed in simultaneously.  Is counting, but all locks must
// be paired with ReadUnlock.  You may get a ReadLock while holding
// a WriteLock.  May only be called from inside a Goth thread
func (lock *goetheLock) ReadLock() error {
	_, err := lock.TryReadLock(-1)
	return err
}

func (lock *goetheLock) TryReadLock(d time.Duration) (bool, error) {
	if d < -1 {
		return false, ErrTryLockDurationIllegal
	}

	if d != -1 && d != 0 {
		panic("implement TryReadLock")
	}

	tid := lock.parent.GetThreadID()
	if tid < 0 {
		return false, ErrNotGoetheThread
	}

	lock.goMux.Lock()
	defer lock.goMux.Unlock()

	if lock.holdingWriter == tid {
		// We can go ahead and increment our count and leave
		lock.incrementReadLock(tid)
		return true, nil
	}

	for lock.holdingWriter >= 0 || lock.writersWaiting > 0 {
		if d == 0 {
			return false, nil
		}

		lock.cond.Wait()
	}

	// At this point holdingWriter < 0 and there are no writersWaiting
	lock.incrementReadLock(tid)

	return true, nil
}

func (lock *goetheLock) incrementReadLock(tid int64) {
	currentValue, found := lock.readerCounts[tid]
	if found {
		currentValue++
		lock.readerCounts[tid] = currentValue
	} else {
		lock.readerCounts[tid] = 1
	}
}

// ReadUnlock unlocks the read lock.  Will only truly leave
// critical section as reader when count is zero
func (lock *goetheLock) ReadUnlock() error {
	tid := lock.parent.GetThreadID()
	if tid < 0 {
		return ErrNotGoetheThread
	}

	lock.goMux.Lock()
	defer lock.goMux.Unlock()

	count, found := lock.readerCounts[tid]
	if !found {
		// Weird case, probably a horrible error
		return nil
	}

	count--
	if count <= 0 {
		delete(lock.readerCounts, tid)

		if lock.writersWaiting > 0 {
			lock.cond.Broadcast()
		}
	} else {
		lock.readerCounts[tid] = count
	}

	return nil
}

// getAllOtherReadCount must have mutex held
func (lock *goetheLock) getAllOtherReadCount(localTid int64) int32 {
	var result int32

	for tid, count := range lock.readerCounts {
		if localTid == -1 || tid != localTid {
			result += count
		}
	}

	return result
}

// getMyReadCount must have mutex held
func (lock *goetheLock) getMyReadCount(localTid int64) int32 {
	value, found := lock.readerCounts[localTid]
	if !found {
		return 0
	}

	return value
}

// WriteLock Locks for write.  Only one writer is allowed
// into the critical section.  Once a WriteLock is requested
// no more readers will be allowed into the critical section
// Is a counting lock.  May only be called from inside a Goth thread
func (lock *goetheLock) WriteLock() error {
	_, err := lock.TryWriteLock(-1)
	return err
}

func (lock *goetheLock) TryWriteLock(d time.Duration) (bool, error) {
	if d < -1 {
		return false, ErrTryLockDurationIllegal
	}

	if d != -1 && d != 0 {
		panic("implement TryWriteLock")
	}

	tid := lock.parent.GetThreadID()
	if tid < 0 {
		return false, ErrNotGoetheThread
	}

	lock.goMux.Lock()
	defer lock.goMux.Unlock()

	if lock.getMyReadCount(tid) != 0 {
		return false, ErrReadLockHeld
	}

	if lock.holdingWriter == tid {
		// counting
		lock.writerCount++
		return true, nil
	}

	lock.writersWaiting++
	for lock.holdingWriter >= 0 || lock.getAllOtherReadCount(tid) > 0 {
		if d == 0 {
			lock.writersWaiting--
			return false, nil
		}

		lock.cond.Wait()
	}

	// I just got this lock for myself
	lock.holdingWriter = tid

	lock.writerCount = 1
	lock.writersWaiting--
	return true, nil
}

// WriteUnlock unlocks write lock.  Will only truly leave
// critical section as reader when count is zero
func (lock *goetheLock) WriteUnlock() error {
	tid := lock.parent.GetThreadID()
	if tid < 0 {
		return ErrNotGoetheThread
	}

	lock.goMux.Lock()
	defer lock.goMux.Unlock()

	if tid != lock.holdingWriter {
		return ErrWriteLockNotHeld
	}

	lock.writerCount--
	if lock.writerCount <= 0 {
		lock.writerCount = 0
		lock.holdingWriter = -2

		lock.cond.Broadcast()
	}

	return nil
}

func (lock *goetheLock) IsLocked() bool {
	return lock.IsReadLocked() || lock.IsWriteLocked()
}

func (lock *goetheLock) IsWriteLocked() bool {
	tid := lock.parent.GetThreadID()
	if tid < 0 {
		channel := make(chan bool)

		lock.parent.Go(lock.channelWriteLocked, channel)

		return <-channel
	}

	return lock.internalWriteLocked()
}

func (lock *goetheLock) channelWriteLocked(retChan chan bool) {
	retChan <- lock.internalWriteLocked()
}

func (lock *goetheLock) internalWriteLocked() bool {
	lock.goMux.Lock()
	defer lock.goMux.Unlock()

	if lock.holdingWriter == -2 {
		return false
	}

	return true
}

func (lock *goetheLock) IsReadLocked() bool {
	tid := lock.parent.GetThreadID()
	if tid < 0 {
		rv := make(chan bool)

		lock.parent.Go(lock.channelIsReadLocked, rv)

		return <-rv
	}

	return lock.internalIsReadLocked()
}

func (lock *goetheLock) channelIsReadLocked(retVal chan bool) {
	retVal <- lock.internalIsReadLocked()
}

func (lock *goetheLock) internalIsReadLocked() bool {
	lock.goMux.Lock()
	defer lock.goMux.Unlock()

	count := lock.getAllOtherReadCount(-1)
	if count > 0 {
		return true
	}

	return false
}
