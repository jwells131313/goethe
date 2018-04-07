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

package main

import (
	"fmt"
	ethe "github.com/jwells131313/goethe"
	"github.com/jwells131313/goethe/utilities"
	"math/rand"
	"time"
)

type poolExample struct {
	lock      ethe.Lock
	jobCount  int
	totalJobs int
}

func useAPool() error {
	goethe := utilities.GetGoethe()

	finished := make(chan bool)
	errors := goethe.NewErrorQueue(1000)

	goethe.Go(func() {
		poolInstance := &poolExample{
			lock: goethe.NewGoetheLock(),
		}

		queue := goethe.NewBoundedFunctionQueue(1000)
		pool, err := goethe.NewPool("example", 5, 10, 5*time.Minute, queue, errors)
		if err != nil {
			finished <- false
			return
		}

		nTime := time.Now()
		unixNanos := nTime.UnixNano()

		source := rand.NewSource(unixNanos)
		rand := rand.New(source)

		err = pool.Start()
		if err != nil {
			finished <- false
			return
		}

		for lcv := 0; lcv < 10; lcv++ {
			poolInstance.incrementJobs()
			queue.Enqueue(poolInstance.randomWork, rand)
		}

		for {
			poolInstance.lock.ReadLock()
			currentJobs := poolInstance.jobCount
			poolInstance.lock.ReadUnlock()

			if currentJobs <= 0 {
				poolInstance.lock.ReadLock()
				fmt.Println("Performed a total of ", poolInstance.totalJobs, " jobs")
				poolInstance.lock.ReadUnlock()

				finished <- true
				return
			}

			time.Sleep(time.Second)
		}

		return
	})

	result := <-finished

	var errorCount int
	for !errors.IsEmpty() {
		errorCount++
		errorInfo, _ := errors.Dequeue()
		if errorInfo != nil {
			fmt.Println("Thread ", errorInfo.GetThreadID(), " returned an error: ", errorInfo.GetError().Error())
		}
	}

	if result {
		fmt.Println("Ended with success along with ", errorCount, " errors")
	} else {
		fmt.Println("Ended with failure along with ", errorCount, "errors")
	}

	return nil
}

func getRandomWorkTime(rand *rand.Rand) time.Duration {
	return time.Duration(rand.Uint32()) % 100
}

func (poolInstance *poolExample) incrementJobs() {
	poolInstance.lock.WriteLock()
	defer poolInstance.lock.WriteUnlock()

	poolInstance.jobCount++

	poolInstance.incrementTotalJobs()
}

func (poolInstance *poolExample) incrementTotalJobs() {
	poolInstance.lock.WriteLock()
	defer poolInstance.lock.WriteUnlock()

	poolInstance.totalJobs++
}

func (poolInstance *poolExample) decrementJobs() {
	poolInstance.lock.WriteLock()
	defer poolInstance.lock.WriteUnlock()

	poolInstance.jobCount--
}

func (poolInstance *poolExample) randomWork(rand *rand.Rand) error {
	defer poolInstance.decrementJobs()

	goethe := utilities.GetGoethe()
	pool, _ := goethe.GetPool("example")

	waitTime := getRandomWorkTime(rand)
	if waitTime%13 == 0 {
		return fmt.Errorf("Failed because we got a wait time of %d milliseconds", waitTime)
	}

	if waitTime%7 == 0 {
		return nil
	}

	time.Sleep(waitTime * time.Millisecond)

	queue := pool.GetFunctionQueue()

	// Keep going
	poolInstance.incrementJobs()
	queue.Enqueue(poolInstance.randomWork, rand)

	return nil
}
