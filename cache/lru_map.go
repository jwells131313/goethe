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

// LRUKeyMap is an LRU map that only keeps the keys (used in the CAR algorithm
// for lists B1 and B2)
type LRUKeyMap interface {
	// AddMRU adds the given key as the MRU of the list
	AddMRU(interface{})
	// RemoveLRU removes the LRU of the list
	RemoveLRU()
	// GetCurrentSize returns the current number of entries in the LRUKeyMap
	GetCurrentSize() int
	// Contains returns true if the given key is in the LRUKeyMap
	Contains(interface{}) bool
}

// NewLRUKeyMap returns an implementation of LRUKeyMap
func NewLRUKeyMap() LRUKeyMap {
	return &lruKeyMapData{
		keys: make(map[interface{}]interface{}),
		list: make([]interface{}, 0),
	}
}

type lruKeyMapData struct {
	lock sync.Mutex
	keys map[interface{}]interface{}
	list []interface{}
}

func (lkm *lruKeyMapData) AddMRU(key interface{}) {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	_, found := lkm.keys[key]
	if found {
		panic("Should never add a key that is already in the list")
	}

	lkm.list = append(lkm.list, key)
	lkm.keys[key] = key
}

func (lkm *lruKeyMapData) RemoveLRU() {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	if len(lkm.list) == 0 {
		return
	}

	removeMe := lkm.list[0]
	delete(lkm.keys, removeMe)

	newSmallerList := make([]interface{}, len(lkm.list)-1)
	if len(lkm.list) > 1 {
		copy(newSmallerList, lkm.list[1:])
	}

	lkm.list = newSmallerList

	return
}

func (lkm *lruKeyMapData) GetCurrentSize() int {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	return len(lkm.list)
}

func (lkm *lruKeyMapData) Contains(key interface{}) bool {
	lkm.lock.Lock()
	defer lkm.lock.Unlock()

	_, found := lkm.keys[key]
	return found
}
