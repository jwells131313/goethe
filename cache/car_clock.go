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

type carClock interface {
	// AddTail always adds to the tail with false page reference
	AddTail(interface{}, interface{})
	Get(interface{}) (interface{}, bool)
	Size() int
	GetPageReferenceOfHead() bool
	RemoveHead() (interface{}, interface{}, bool)
	SetPageReference(interface{})
	RemoveAll(func(key interface{}, value interface{}) bool)
}

type carClockValue struct {
	key              interface{}
	value            interface{}
	pageReferenceBit bool
	next             *carClockValue
}

type carClockData struct {
	lock   sync.Mutex
	keyMap map[interface{}]*carClockValue
	head   *carClockValue
	tail   *carClockValue
}

func newCarClock() carClock {
	return &carClockData{
		keyMap: make(map[interface{}]*carClockValue),
	}
}

func (ccd *carClockData) AddTail(key interface{}, value interface{}) {
	ccd.lock.Lock()
	defer ccd.lock.Unlock()

	_, found := ccd.keyMap[key]
	if found {
		panic("cannot add already existing key to clock")
	}

	addMe := &carClockValue{
		key:   key,
		value: value,
	}

	if ccd.tail == nil {
		ccd.head = addMe
		ccd.tail = addMe
	} else {
		ccd.tail.next = addMe
		ccd.tail = addMe
	}

	ccd.keyMap[key] = addMe
}

func (ccd *carClockData) Get(key interface{}) (interface{}, bool) {
	ccd.lock.Lock()
	defer ccd.lock.Unlock()

	ccv, found := ccd.keyMap[key]
	if !found {
		return nil, false
	}

	return ccv.value, true
}

func (ccd *carClockData) Size() int {
	ccd.lock.Lock()
	defer ccd.lock.Unlock()

	return len(ccd.keyMap)
}

func (ccd *carClockData) GetPageReferenceOfHead() bool {
	ccd.lock.Lock()
	defer ccd.lock.Unlock()

	if ccd.head == nil {
		panic("can not be called on an empty carClock")
	}

	return ccd.head.pageReferenceBit
}

func (ccd *carClockData) RemoveHead() (interface{}, interface{}, bool) {
	ccd.lock.Lock()
	defer ccd.lock.Unlock()

	toRemove := ccd.head
	if toRemove == nil {
		return nil, nil, false
	}

	delete(ccd.keyMap, toRemove.key)

	ccd.head = toRemove.next
	if ccd.head == nil {
		ccd.tail = nil
	}

	return toRemove.key, toRemove.value, true
}

func (ccd *carClockData) SetPageReference(key interface{}) {
	ccd.lock.Lock()
	defer ccd.lock.Unlock()

	toUpdate, found := ccd.keyMap[key]
	if !found {
		return
	}

	toUpdate.pageReferenceBit = true
}

func (ccd *carClockData) RemoveAll(rf func(key interface{}, value interface{}) bool) {
	ccd.lock.Lock()
	defer ccd.lock.Unlock()

	if ccd.head == nil {
		return
	}

	dot := ccd.head
	var prior *carClockValue
	for {
		val := ccd.keyMap[dot.key]
		if rf(dot.key, val) {
			if prior == nil {
				ccd.head = dot.next
			} else {
				prior.next = dot.next
			}

			if dot.next == nil {
				ccd.tail = prior
			}

			delete(ccd.keyMap, dot.key)
		}

		prior = dot
		dot = dot.next
		if dot == nil {
			return
		}
	}

}
