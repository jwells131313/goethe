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

import "sync"

// lruKeyMap is an LRU map that only keeps the keys (used in the CAR algorithm
// for lists B1 and B2)
type lruKeyMap interface {
	// AddMRU adds the given key as the MRU of the list
	AddMRU(interface{})
	// RemoveLRU removes the LRU of the list
	RemoveLRU()
	// Remove removes a specific key from the cache
	Remove(interface{})
	// GetCurrentSize returns the current number of entries in the LRUKeyMap
	GetCurrentSize() int
	// Contains returns true if the given key is in the LRUKeyMap
	Contains(interface{}) bool
}

type lruDoubleLinkedData struct {
	key      interface{}
	previous *lruDoubleLinkedData
	next     *lruDoubleLinkedData
}

type lruKeyMapData struct {
	lock sync.Mutex
	keys map[interface{}]*lruDoubleLinkedData
	mru  *lruDoubleLinkedData
	lru  *lruDoubleLinkedData
}

// newLRUKeyMap returns an implementation of LRUKeyMap
func newLRUKeyMap() lruKeyMap {
	return &lruKeyMapData{
		keys: make(map[interface{}]*lruDoubleLinkedData),
	}
}

func (lkm *lruKeyMapData) AddMRU(key interface{}) {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	_, found := lkm.keys[key]
	if found {
		panic("Should never add a key that is already in the list")
	}

	addMe := &lruDoubleLinkedData{
		key:  key,
		next: lkm.mru,
	}

	if lkm.mru == nil {
		lkm.mru = addMe
		lkm.lru = addMe
	} else {
		lkm.mru.previous = addMe
		lkm.mru = addMe
	}

	lkm.keys[key] = addMe
}

func (lkm *lruKeyMapData) RemoveLRU() {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	if lkm.lru == nil {
		return
	}

	delete(lkm.keys, lkm.lru.key)

	if lkm.lru.previous == nil {
		lkm.mru = nil
		lkm.lru = nil
	} else {
		lkm.lru.previous.next = nil
		lkm.lru = lkm.lru.previous
	}

	return
}

func (lkm *lruKeyMapData) Remove(key interface{}) {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	node, found := lkm.keys[key]
	if !found {
		return
	}
	delete(lkm.keys, key)

	if node.previous != nil {
		node.previous.next = node.next
	} else {
		// the mru needs to move
		lkm.mru = node.next
	}

	if node.next != nil {
		node.next.previous = node.previous
	} else {
		// the lru needs to move
		lkm.lru = node.previous
	}
}

func (lkm *lruKeyMapData) GetCurrentSize() int {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	return len(lkm.keys)
}

func (lkm *lruKeyMapData) Contains(key interface{}) bool {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	_, found := lkm.keys[key]
	return found
}
