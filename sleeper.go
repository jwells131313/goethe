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
	"fmt"
	"github.com/jwells131313/goethe/queues"
	"io"
	"sync"
	"time"
)

type sleeper interface {
	// sleep with the amount of time to wait, the condition to ring when done, and the job number
	sleep(time.Duration, *sync.Cond, uint64) io.Closer
}

type sleeperNode struct {
	ringTime *time.Time
	cond     *sync.Cond
	id       uint64
	closed   bool
}

type sleeperImpl struct {
	heap queues.Heap
	lock sync.Mutex
	jobs map[uint64]uint64
}

func newSleeper() sleeper {
	return &sleeperImpl{
		heap: queues.NewHeap(sleeperComparator),
		jobs: make(map[uint64]uint64),
	}
}

func (sleepy *sleeperImpl) sleep(duration time.Duration, cond *sync.Cond, jobNumber uint64) io.Closer {
	sleepy.lock.Lock()
	defer sleepy.lock.Unlock()

	if duration < fudgeFactor {
		cond.Broadcast()
		return nil
	}

	_, has := sleepy.jobs[jobNumber]
	if has {
		return nil
	}

	sleepy.jobs[jobNumber] = jobNumber

	ringsAt := time.Now().Add(duration)
	newNode := &sleeperNode{
		ringTime: &ringsAt,
		cond:     cond,
		id:       jobNumber,
	}

	startNewThread := true

	raw, found := sleepy.heap.Peek()
	if found {
		sn := raw.(*sleeperNode)
		nextNode := sn.ringTime

		until := nextNode.Sub(ringsAt)

		if until < 0 {
			// start new thread
			startNewThread = false
		}
	}

	sleepy.heap.Add(newNode)

	if startNewThread {
		GetGoethe().Go(sleepy.waiter, duration)
	}

	return newNode
}

func (sleepy *sleeperImpl) waiter(duration time.Duration) {
	time.Sleep(duration)

	sleepy.lock.Lock()
	defer sleepy.lock.Unlock()

	raw, found := sleepy.heap.Peek()
	if !found {
		return
	}

	sn := raw.(*sleeperNode)
	fireTime := sn.ringTime

	nextFire := time.Until(*fireTime)
	for nextFire < fudgeFactor {
		// remove peeked node
		sleepy.heap.Get()

		delete(sleepy.jobs, sn.id)

		if !sn.closed {
			sn.cond.Broadcast()
		}

		raw, found = sleepy.heap.Peek()
		if !found {
			return
		}

		sn = raw.(*sleeperNode)
		fireTime = sn.ringTime

		nextFire = time.Until(*fireTime)
	}
}

func (node *sleeperNode) Close() error {
	node.closed = true
	return nil
}

func (node *sleeperNode) String() string {
	return fmt.Sprintf("SleeperNode(%v, %d)", node.ringTime, node.id)
}

func sleeperComparator(aRaw interface{}, bRaw interface{}) int {
	a := aRaw.(*sleeperNode)
	b := bRaw.(*sleeperNode)

	aTime := a.ringTime
	bTime := b.ringTime

	if bTime.After(*aTime) {
		return 1
	}
	if aTime.After(*bTime) {
		return -1
	}

	return 0
}
