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

package internal

import (
	"github.com/jwells131313/goethe"
	"sync"
	"time"
)

type functionErrorQueue struct {
	mux     sync.Mutex
	cond    *sync.Cond
	changer func(queue goethe.FunctionQueue)

	capacity uint32
	queue    []*goethe.FunctionDescriptor
}

// NewFunctionQueue creates a new function queue with the given capacity
func NewFunctionQueue(userCapacity uint32) goethe.FunctionQueue {
	retVal := &functionErrorQueue{
		capacity: userCapacity,
		queue:    make([]*goethe.FunctionDescriptor, 0),
	}

	retVal.cond = sync.NewCond(&retVal.mux)

	return retVal
}

func (fq *functionErrorQueue) Enqueue(userCall interface{}, args ...interface{}) error {
	if userCall == nil {
		return nil
	}

	fq.mux.Lock()
	defer fq.mux.Unlock()

	if uint32(len(fq.queue)) >= fq.capacity {
		return goethe.ErrAtCapacity
	}

	descriptor := &goethe.FunctionDescriptor{
		UserCall: userCall,
		Args:     make([]interface{}, len(args)),
	}

	for index, arg := range args {
		descriptor.Args[index] = arg
	}

	fq.queue = append(fq.queue, descriptor)

	fq.cond.Broadcast()
	if fq.changer != nil {
		go fq.changer(fq)
	}

	return nil
}

func (fq *functionErrorQueue) Dequeue(duration time.Duration) (*goethe.FunctionDescriptor, error) {
	fq.mux.Lock()
	defer fq.mux.Unlock()

	currentTime := time.Now()
	elapsedDuration := time.Since(currentTime)

	for (duration > 0) && (elapsedDuration < duration) && (len(fq.queue) <= 0) {
		timer := time.AfterFunc(duration-elapsedDuration, func() {
			fq.cond.Broadcast()
		})

		fq.cond.Wait()

		timer.Stop()

		elapsedDuration = time.Since(currentTime)
	}

	if len(fq.queue) <= 0 {
		return nil, goethe.ErrEmptyQueue
	}

	retVal := fq.queue[0]
	fq.queue = fq.queue[1:]

	if fq.changer != nil {
		go fq.changer(fq)
	}

	return retVal, nil
}

func (fq *functionErrorQueue) GetCapacity() uint32 {
	return fq.capacity
}

func (fq *functionErrorQueue) GetSize() int {
	fq.mux.Lock()
	defer fq.mux.Unlock()

	return len(fq.queue)
}

func (fq *functionErrorQueue) IsEmpty() bool {
	return fq.GetSize() <= 0
}

func (fq *functionErrorQueue) SetStateChangeCallback(ch func(goethe.FunctionQueue)) {
	fq.mux.Lock()
	defer fq.mux.Unlock()

	fq.changer = ch
}
