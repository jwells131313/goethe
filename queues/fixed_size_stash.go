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

package queues

import "time"

// StashCreateFunction is the function that will be called, possibly in
// separate threads, to create an entity in the FixedSizeStash
type StashCreateFunction func() (interface{}, error)

// FixedSizeStash is a stash of items that should be kept at a certain size
// When an item is removed from the stash with Get a new item will be added
// to the stash in the background with the create function.  Using a stash
// like this is useful when the creation time of an object is very high but
// many of them may be needed very quickly but sporadically.  The size of the
// stash should be determined by how high the expected peak request rate may become.
// For example, if it takes five minutes to create one of the items in the
// stash and its known that once an hour a burst of 15 requests come in at
// the same time it would be good to set the size to 15 to handle all of those
// requests quickly.  The system will run the StashCreateFunction
type FixedSizeStash interface {
	// GetDesiredSize Returns the desired size of the fixed size stash
	GetDesiredSize() int
	// GetCurrentSize Returns the current size of the fixed size stash (how many are currently available)
	GetCurrentSize() int
	// GetCreateFunction Returns the create function used to create elements of the stash
	GetCreateFunction() StashCreateFunction
	// GetErrorChannel returns the error channel for when the create function fails
	GetErrorChannel() <-chan error
	// Get returns an element from the stash, or false if there were no elements available
	Get() (interface{}, bool)
	// WaitForElement returns an element from the stash, and will block until an element becomes available
	// A duration of zero will return immediately with an available element or an error.  A negative duration
	// will return an error immediately without looking for an element.
	WaitForElement(howLong time.Duration) (interface{}, error)
}

type fixedSizeStashData struct {
	creator      StashCreateFunction
	desiredSize  int
	errorChannel chan error
}

func NewFixedSizeStash(creator StashCreateFunction, desiredSize int, eChan chan error) FixedSizeStash {
	return &fixedSizeStashData{
		creator:      creator,
		desiredSize:  desiredSize,
		errorChannel: eChan,
	}

}

func (fssd *fixedSizeStashData) GetDesiredSize() int {
	return fssd.desiredSize
}

func (fssd *fixedSizeStashData) GetCurrentSize() int {
	panic("implement me")
}

func (fssd *fixedSizeStashData) GetCreateFunction() StashCreateFunction {
	return fssd.creator
}

func (fssd *fixedSizeStashData) GetErrorChannel() <-chan error {
	return fssd.errorChannel
}

func (fssd *fixedSizeStashData) Get() (interface{}, bool) {
	r, e := fssd.WaitForElement(0)
	if e != nil {
		return nil, false
	}
	return r, true
}

func (fssd *fixedSizeStashData) WaitForElement(howLong time.Duration) (interface{}, error) {
	panic("implement me")
}
