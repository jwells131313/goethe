[//]: # " DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER. "
[//]: # "  "
[//]: # " Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved. "
[//]: # "  "
[//]: # " The contents of this file are subject to the terms of either the GNU "
[//]: # " General Public License Version 2 only (''GPL'') or the Common Development "
[//]: # " and Distribution License(''CDDL'') (collectively, the ''License'').  You "
[//]: # " may not use this file except in compliance with the License.  You can "
[//]: # " obtain a copy of the License at "
[//]: # " https://oss.oracle.com/licenses/CDDL+GPL-1.1 "
[//]: # " or LICENSE.txt.  See the License for the specific "
[//]: # " language governing permissions and limitations under the License. "
[//]: # "  "
[//]: # " When distributing the software, include this License Header Notice in each "
[//]: # " file and include the License file at LICENSE.txt. "
[//]: # "  "
[//]: # " GPL Classpath Exception: "
[//]: # " Oracle designates this particular file as subject to the ''Classpath'' "
[//]: # " exception as provided by Oracle in the GPL Version 2 section of the License "
[//]: # " file that accompanied this code. "
[//]: # "  "
[//]: # " Modifications: "
[//]: # " If applicable, add the following below the License Header, with the fields "
[//]: # " enclosed by brackets [] replaced by your own identifying information: "
[//]: # " ''Portions Copyright [year] [name of copyright owner]'' "
[//]: # "  "
[//]: # " Contributor(s): "
[//]: # " If you wish your version of this file to be governed by only the CDDL or "
[//]: # " only the GPL Version 2, indicate your decision by adding ''[Contributor] "
[//]: # " elects to include this software in this distribution under the [CDDL or GPL "
[//]: # " Version 2] license.''  If you don't indicate a single choice of license, a "
[//]: # " recipient has the option to distribute your version of this file under "
[//]: # " either the CDDL, the GPL Version 2 or to extend the choice of license to "
[//]: # " its licensees as provided above.  However, if you add GPL Version 2 code "
[//]: # " and therefore, elected the GPL Version 2 license, then the option applies "
[//]: # " only if the new code is made subject to such option by the copyright "
[//]: # " holder. "

# goethe [![GoDoc](https://godoc.org/github.com/jwells131313/goethe?status.svg)](https://godoc.org/github.com/jwells131313/goethe) [![wercker status](https://app.wercker.com/status/269d55f515cfd042ec505993f5e3fe88/s/master "wercker status")](https://app.wercker.com/project/byKey/269d55f515cfd042ec505993f5e3fe88) [![Go Report Card](https://goreportcard.com/badge/github.com/jwells131313/goethe)](https://goreportcard.com/report/github.com/jwells131313/goethe)

Threading utilities for GO

## Usage

This package provides several useful threading and locking utilities for use in golang.  In
particular goethe (pronounced ger-tay) threads have thread-ids which can be useful for logging
and in thread pools.

This package maintains one global goethe implementation which can be gotten using the
github.com/jwells131313/goethe.GetGoethe() method.

NOTE:  The current version of this API is 0.1.  This means that the API has
not settled completely and may change in future revisions.  Once the goethe
team has decided the API is good as it is we will make the 1.0 version which
will have some backward compatibility guarantees.  In the meantime, if you
have questions or comments please open issues.  Thank you.

1. [ThreadID](#ThreadID)
2. [Recursive Locks](#recursive-locks)
3. [Thread Pools](#thread-pools)
4. [Thread Local Storage](#thread-local-storage)
5. [Timers](#timers)

### ThreadID

Inside a goethe thread you can use the utilities.GetGoethe() method to get the implementation of Goethe and
use that to get the ID of the thread in which the code is currently running:

```go
package foo

import (
	"fmt"
	"github.com/jwells131313/goethe"
)

func basic() {
	ethe := goethe.GetGoethe()

	channel := make(chan int64)

	ethe.Go(func() {
		// A thread ID!
		tid := goethe.GetThreadID()

		// Tell momma about our thread-id
		channel <- tid
	})

	threadID := <-channel
	fmt.Println("the threadID in my thread was ", threadID)
}
```

You can also use a method that takes arguments with goethe.  For this
use the GoWithArgs method as in the following example:

```go
package foo

import (
	"fmt"
	"github.com/jwells131313/goethe"
)

func addMe(a, b, c int, ret chan int) {
	ret <- a + b + c
}

func basicWithArgs() {
	ethe := goethe.GetGoethe()

	channel := make(chan int)

	ethe.GoWithArgs(addMe, 1, 2, 3, channel)

	sum := <-channel
	fmt.Println("the sum in my thread was ", sum)
}
```

### Recursive Locks

In goethe threads you can have recursive reader/write mutexes which obey the following rules:

* Only one writer lock is allowed into the critical section
* When holding a writer lock on a thread another writer lock may be acquired on the same thread (counting)
* When holding a writer lock on a thread you may acquire a reader lock on the same thread.  The writer lock remains in effect
* Many reader locks can be held on multiple different threads
* When holding a reader lock on a thread another reader lock may be acquired on the same thread (counting)
* When holding a writer lock you may not acquire a reader lock.  Doing so leads too easily to deadlocks
* Once a writer asks for the lock no more readers will be able to enter, so writers can starve readers

The following is an example of a recursive write lock
```go
package main

import (
	"github.com/jwells131313/goethe"
)

var ethe goethe.Goethe = goethe.GetGoethe()
var lock goethe.Lock = ethe.NewGoetheLock()

func writer1() {
	lock.WriteLock()
	defer lock.WriteUnlock()

	writer2()
}

func writer2() {
	lock.WriteLock()
	defer lock.WriteUnlock()
}

func main() {
	ethe.Go(writer1)
}
```

If you try to call a Lock or Unlock method of a goethe lock while not inside a goethe thread it
will return an error.

### Thread Pools

Thread pools use goethe threads so that you can use thread-ids, thread-locals and recursive
locks in your threaded application.

In order to create a thread pool you specify the min threads in the the pool, the max threads
in the pool and the decay time, which is the time a thread will be idle before being released
(if the number of threads is greater than the min).

You also give the pool an implementation of a FunctionQueue.  You can also get a default
implementation of FunctionQueue from the goethe service.  However any implementation of
FunctionQueue will work, which allows you to use priority queues or other queue implementations.
Once a queue is associated with a pool you use the queue Enqueue API to give jobs to the thread pool.

You can also give the pool an ErrorQueue.  Any non-nil errors returned from the enqueued jobs will
be placed on the ErrorQueue.  It is up to the application to check and drain the ErrorQueue for
errors.

The following example uses recursive read/write locks, an error queue and a functional queue along
with a pool.  The actual work done in the randomWork method is just sleeping anywhere from 1 to 99
milliseconds.  However, if the number of milliseconds to sleep is divisible by 13 then the randomWork
method will return with an error.  If the number of milliseconds to sleep is divisible by seven
then randomWork will exit with no error but will not put new work on the queue.  Otherwise random
work sleeps the amount of time and then adds itself back on the functional queue.  The main
subroutine called useAPool initializes the pool and starts it and then waits for all the randomWork
jobs to finish.  The counters to determine if all jobs are finished are protected with read/write
locks.  Here is the example:

```go
type poolExample struct {
	lock      goethe.Lock
	jobCount  int
	totalJobs int
}

func useAPool() error {
	ethe := goethe.GetGoethe()

	finished := make(chan bool)
	errors := goethe.NewErrorQueue(1000)

	ethe.Go(func() {
		poolInstance := &poolExample{
			lock: goethe.NewGoetheLock(),
		}

		queue := goethe.NewBoundedFunctionQueue(1000)
		pool, err := ethe.NewPool("example", 5, 10, 5*time.Minute, queue, errors)
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

	ethe := goethe.GetGoethe()
	pool, _ := ethe.GetPool("example")

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
```

One thing to notice is that use of the recursive writeLock is made safely and correctly!  No
critical sections were harmed in the making of this example!

### Thread Local Storage

Goethe threads can take advantage of named thread local storage.  Thread local storage is first
established by giving it a name, an initializer function and a destroyer function.  Then
in your goethe threads you can call GetThreadLocal with the name of your thread local and get
the type returned by the initializer.  Each thread has its own copy of the type so two threads
will not interfere with each other.  When the thread goes away the destructor for all named
thread local storage associated with that thread will be called.

Under construction: need an example of the use of thread local storage

### Timers

Goethe provides timer threads that run user code periodically.  There are two types of timers, one
with fixed delay and one with fixed rate.

A fixed delay timer will start the next timer once the user code has run to completion.  So
long running user code will cause the timer to not be invoked again until the user code has
completed and the delay period has passed again.  This timer is started with the
Goethe.ScheduleWithFixedDelay method

A fixed rate timer will run every multiple of the given period, whether or not the user code
from previous runs of the timer have completed (on different Goethe threads).  This
allows for stricter control of the scheduling of the timer, but the user code must be written
with the knowledge that it could be run again on another thread.

Both timers set a ThreadLocalStorage variable named **goethe.Timer** (also held in the
goethe.TimerThreadLocal variable).  This makes it easy to give the running timer code the
ability to cancel the timer itself.

In the following example a bell is run every five minutes 12 times, after which time the
thread itself cancels the timer.

```go
var count int

// RunClockForOneHour ringing a bell every five minutes
func RunClockForOneHour() {
	ethe := utilities.GetGoethe()

	ethe.ScheduleAtFixedRate(0, 5*time.Minute, nil, ringBell)
}

func ringBell() {
	count++
	if count >= 12 {
		ethe := utilities.GetGoethe()

		tl, _ := ethe.GetThreadLocal(goethe.TimerThreadLocal)
		i, _ := tl.Get()
		timer := i.(goethe.Timer)
		timer.Cancel()
	}

	fmt.Printf("Bell at count %d\a\n", count)
}
```

### Under Construction

In the future it is intended for goethe to provide the following:

* locks you can try
* locks you can give up on after some duration

![](https://github.com/jwells131313/goethe/blob/master/images/goth.jpg "Go Thread Utilities")
