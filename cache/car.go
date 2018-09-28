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

import "github.com/jwells131313/goethe"

type carCache struct {
	lock         goethe.Lock
	calculator   Computable
	cycleHandler CycleHandler
	max          int
	p            int
	T1           carClock
	T2           carClock
	B1           lruKeyMap
	B2           lruKeyMap
}

// NewCARCache creates a compute-function in-memory cache where the values
// are only ever instantiated once.  The maximum size of the values held in the cache is given
// by max.  This cache uses the CAR algorithm to determine which keys are removed from the
// cache when space is exhausted.
//
// The CAR algorithm keeps at most max values but up to 2 * max the number of keys
func NewCARCache(max int, calculator Computable, cycleHandler CycleHandler) (Cache, error) {
	return &carCache{
		lock:         gd.NewGoetheLock(),
		calculator:   calculator,
		cycleHandler: cycleHandler,
		max:          max,
		T1:           newCarClock(),
		T2:           newCarClock(),
		B1:           newLRUKeyMap(),
		B2:           newLRUKeyMap(),
	}, nil
}

func (cc *carCache) Compute(key interface{}) (interface{}, error) {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan (*onGoetheReply))

		gd.Go(cc.channelInternalCompute, key, replyChan)

		reply := <-replyChan

		return reply.value, reply.err
	}

	return cc.internalCompute(key)
}

func (cc *carCache) GetCalculator() Computable {
	return cc.calculator
}

func (cc *carCache) GetCycleHandler() CycleHandler {
	return cc.cycleHandler
}

func (cc *carCache) HasKey(key interface{}) bool {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan bool)

		gd.Go(cc.channelHasKey, key, replyChan)

		reply := <-replyChan

		return reply
	}

	return cc.internalHasKey(key)
}

func (cc *carCache) Clear() {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan bool)

		gd.Go(cc.channelClear, replyChan)

		<-replyChan

		return
	}

	cc.internalClear()
}

func (cc *carCache) Remove(removalFunc func(key interface{}, value interface{}) bool) {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan bool)

		gd.Go(cc.channelRemove, removalFunc, replyChan)

		<-replyChan

		return
	}

	cc.internalRemove(removalFunc)
}

func (cc *carCache) Size() int {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan int)

		gd.Go(cc.channelSize, replyChan)

		retVal := <-replyChan

		return retVal
	}

	return cc.internalSize()
}

func (cc *carCache) channelInternalCompute(key interface{}, reply chan (*onGoetheReply)) {
	value, err := cc.internalCompute(key)

	reply <- &onGoetheReply{
		value: value,
		err:   err,
	}
}

func (cc *carCache) internalCompute(key interface{}) (interface{}, error) {
	cc.lock.WriteLock()
	defer cc.lock.WriteUnlock()

	value, found := cc.T1.Get(key)
	if found {
		// Cache hit
		cc.T1.SetPageReference(key)
		return value, nil
	}

	value, found = cc.T2.Get(key)
	if found {
		// Cache hit
		cc.T2.SetPageReference(key)
		return value, nil
	}

	asks, err := checkCycle(carCacheThreadLocalName, key, cc.cycleHandler)
	if err != nil {
		return nil, err
	}

	asks[key] = key
	defer delete(asks, key)

	// Cache miss
	value, err = cc.calculator.Compute(key)
	if err != nil {
		return nil, err
	}

	totalCacheSize := cc.T1.Size() + cc.T2.Size()
	if totalCacheSize >= cc.max {
		// choose and remove a value from T1 or T2 and put it in B1 or B2
		cc.replace()

		if !cc.isXInB(key) {
			t1Size := cc.T1.Size()
			b1Size := cc.B1.GetCurrentSize()

			if (t1Size + b1Size) >= cc.max {
				cc.B1.RemoveLRU()
			} else {
				t2Size := cc.T2.Size()
				b2Size := cc.B2.GetCurrentSize()

				allKeysSize := t1Size + t2Size + b1Size + b2Size
				if allKeysSize >= (2 * cc.max) {
					cc.B2.RemoveLRU()
				}

			}

		}
	}

	if !cc.isXInB(key) {
		cc.T1.AddTail(key, value)
	} else {
		b1Size := cc.B1.GetCurrentSize()
		b2Size := cc.B2.GetCurrentSize()

		if cc.B1.Contains(key) {
			newP := min(cc.p+max(1, b2Size/b1Size), cc.max)
			cc.p = newP

			cc.B1.Remove(key)

			cc.T2.AddTail(key, value)
		} else {
			newP := max(cc.p-max(1, b1Size/b2Size), 0)
			cc.p = newP

			cc.B2.Remove(key)

			cc.T2.AddTail(key, value)
		}
	}

	return value, nil
}

func (cc *carCache) channelSize(retChan chan int) {
	retVal := cc.internalSize()

	retChan <- retVal
}

func (cc *carCache) internalSize() int {
	cc.lock.ReadLock()
	defer cc.lock.ReadUnlock()

	return cc.T1.Size() + cc.T2.Size()
}

func (cc *carCache) channelClear(retChan chan bool) {
	cc.internalClear()

	retChan <- true
}

func (cc *carCache) internalClear() {
	cc.lock.WriteLock()
	defer cc.lock.WriteUnlock()

	cc.p = 0
	cc.T1 = newCarClock()
	cc.T2 = newCarClock()
	cc.B1 = newLRUKeyMap()
	cc.B2 = newLRUKeyMap()
}

func (cc *carCache) channelHasKey(key interface{}, retChan chan bool) {
	ret := cc.internalHasKey(key)

	retChan <- ret
}

func (cc *carCache) internalHasKey(key interface{}) bool {
	cc.lock.ReadLock()
	defer cc.lock.ReadUnlock()

	_, found := cc.T1.Get(key)
	if found {
		return true
	}

	_, found = cc.T2.Get(key)
	if found {
		return true
	}

	return false
}

func (cc *carCache) channelRemove(removalFunc func(key interface{}, value interface{}) bool, retChan chan bool) {
	cc.internalRemove(removalFunc)

	retChan <- true
}

func (cc *carCache) internalRemove(removalFunc func(key interface{}, value interface{}) bool) {
	cc.lock.WriteLock()
	defer cc.lock.WriteUnlock()

	cc.T1.RemoveAll(removalFunc)
	cc.T2.RemoveAll(removalFunc)
	cc.B1.RemoveAll(removalFunc)
	cc.B2.RemoveAll(removalFunc)
}

func (cc *carCache) isXInB(key interface{}) bool {
	return cc.B1.Contains(key) || cc.B2.Contains(key)
}

func (cc *carCache) replace() {
	localP := max(1, cc.p)

	for {
		t1Size := cc.T1.Size()

		if t1Size >= localP {
			if !cc.T1.GetPageReferenceOfHead() {
				// found in recency
				key, _, _ := cc.T1.RemoveHead()
				cc.B1.AddMRU(key)

				return
			}

			// Promoting to T2
			key, value, _ := cc.T1.RemoveHead()
			cc.T2.AddTail(key, value)

		} else {
			if !cc.T2.GetPageReferenceOfHead() {
				// found in frequency
				key, _, _ := cc.T2.RemoveHead()
				cc.B2.AddMRU(key)

				return
			}

			// Clocks it front to back, but with bit set to false now
			key, value, _ := cc.T2.RemoveHead()
			cc.T2.AddTail(key, value)
		}

	}

}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
