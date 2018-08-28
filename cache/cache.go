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
	"github.com/jwells131313/goethe"
)

var (
	gd = goethe.GG()
)

const (
	cacheThreadLocalName = "GoetheCacheThreadLocal"
)

type cacheData struct {
	lock  goethe.Lock
	cache map[interface{}]interface{}

	calculater   Computable
	cycleHandler CycleHandler
}

type onGoetheReply struct {
	value interface{}
	err   error
}

func init() {
	gd.EstablishThreadLocal(cacheThreadLocalName, func(tl goethe.ThreadLocal) error {
		tl.Set(make(map[interface{}]interface{}))
		return nil
	}, nil)
}

// NewCache creates a cache of key/value pairs where the value can be computed
// directly from the key.  This is a useful cache if the computation of the value
// is difficult or time-consuming in some way and your code would prefer that it
// happens only once.  The computation of the value may involve computing other
// values, in which case cycles might occur.  In these cases you can provide a
// CycleHandler to do whatever remediation is necessary.  Otherwise if cycleHandler
// is nil an error will be thrown when a cycle has been detected.  This method is
// thread safe.  If two threads try to create the same value the value will still
// only be computed once, and the thread that loses the race will use the cached
// value
func NewCache(calculator Computable, cycleHandler CycleHandler) (Cache, error) {
	return &cacheData{
		lock:         gd.NewGoetheLock(),
		cache:        make(map[interface{}]interface{}),
		calculater:   calculator,
		cycleHandler: cycleHandler,
	}, nil
}

func (cache *cacheData) Compute(key interface{}) (interface{}, error) {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan (*onGoetheReply))

		gd.Go(cache.channelInternalCompute, key, replyChan)

		reply := <-replyChan

		return reply.value, reply.err
	}

	return cache.internalCompute(key)
}

func (cache *cacheData) GetCalculator() Computable {
	return cache.calculater
}

func (cache *cacheData) GetCycleHandler() CycleHandler {
	return cache.cycleHandler
}

func (cache *cacheData) HasKey(key interface{}) bool {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan bool)

		gd.Go(cache.channelHasKey, key, replyChan)

		reply := <-replyChan

		return reply
	}

	return cache.internalHasKey(key)
}

func (cache *cacheData) channelHasKey(key interface{}, retChan chan bool) {
	ret := cache.internalHasKey(key)

	retChan <- ret
}

func (cache *cacheData) internalHasKey(key interface{}) bool {
	cache.lock.ReadLock()
	defer cache.lock.ReadUnlock()

	_, found := cache.cache[key]

	return found
}

func (cache *cacheData) Clear() {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan bool)

		gd.Go(cache.channelClear, replyChan)

		<-replyChan

		return
	}

	cache.internalClear()
}

func (cache *cacheData) channelClear(retChan chan bool) {
	cache.internalClear()

	retChan <- true
}

func (cache *cacheData) internalClear() {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.cache = nil
	cache.cache = make(map[interface{}]interface{})
}

func (cache *cacheData) Remove(removalFunc func(key interface{}) bool) {
	tid := gd.GetThreadID()
	if tid < 0 {
		replyChan := make(chan bool)

		gd.Go(cache.channelRemove, removalFunc, replyChan)

		<-replyChan

		return
	}

	cache.internalRemove(removalFunc)
}

func (cache *cacheData) channelRemove(removalFunc func(key interface{}) bool, retChan chan bool) {
	cache.internalRemove(removalFunc)

	retChan <- true
}

func (cache *cacheData) internalRemove(removalFunc func(key interface{}) bool) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	for key, _ := range cache.cache {
		if removalFunc(key) {
			delete(cache.cache, key)
		}
	}
}

func (cache *cacheData) channelInternalCompute(key interface{}, reply chan (*onGoetheReply)) {
	value, err := cache.internalCompute(key)

	reply <- &onGoetheReply{
		value: value,
		err:   err,
	}
}

func (cache *cacheData) internalCompute(key interface{}) (interface{}, error) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	value, found := cache.cache[key]
	if found {
		return value, nil
	}

	// Must compute, but first see if we've tried to find it before
	threadLocal, err := gd.GetThreadLocal(cacheThreadLocalName)
	if err != nil {
		return nil, err
	}

	rawThreadLocal, err := threadLocal.Get()
	if err != nil {
		return nil, err
	}

	asks, ok := rawThreadLocal.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("internal error, thread local not expected type")
	}

	_, found = asks[key]
	if found {
		// Detected a cycle
		if cache.cycleHandler != nil {
			err = cache.cycleHandler(key)
			if err != nil {
				return nil, err
			}
		}

		return nil, fmt.Errorf("A cycle was detected for this key: %v", key)
	}

	asks[key] = key

	// No cycle, now do the compute
	value, err = cache.calculater.Compute(key)
	if err != nil {
		return nil, err
	}

	// Found the value, store it away
	cache.cache[key] = value

	return value, nil
}
