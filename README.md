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

This package provides several useful threading, locking and caching utilities for use in golang.  In
particular goethe (pronounced ger-tay) threads have thread-ids which can be useful for logging
and in thread pools.

This package maintains one global goethe implementation which can be gotten using the
github.com/jwells131313/goethe.GetGoethe() method.

1. [ThreadID](#threadid)
2. [Cache](#in-memory-computable-cache)
3. [LRU-Style CAR Cache](#car-cache)
4. [Heap Queue](#heap)
5. [Recursive Locks](#recursive-locks)
6. [Thread Local Storage](#thread-local-storage)
7. [Timers](#timers)
8. [Thread Pools](#thread-pools)

### ThreadID

Use the GetGoethe() or GG() method to get an implementation of ThreadUtilities and use that to
get the ID of the thread in which the code is currently running:

```go
package foo

import (
	"fmt"
	"github.com/jwells131313/goethe"
)

func basic() {
	ethe := goethe.GG()

	channel := make(chan int64)

	ethe.Go(func() {
		// A thread ID!
		tid := ethe.GetThreadID()

		// Tell momma about our thread-id
		channel <- tid
	})

	threadID := <-channel
	fmt.Println("the threadID in my thread was ", threadID)
}
```

You can also use the ThreadUtilities.Go function for a method that takes arguments.  The following
example passes the parameters into the function.

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

	ethe.Go(addMe, 1, 2, 3, channel)

	sum := <-channel
	fmt.Println("the sum in my thread was ", sum)
}
```

## Caches

### In-Memory Computable Cache

[![GoDoc](https://godoc.org/github.com/jwells131313/goethe/cache?status.svg)](https://godoc.org/github.com/jwells131313/goethe/cache)

The cache package contains methods to create an in-memory computable cache.  A computable cache
is a cache where the values can be computed directly from the keys.  A computable cache is useful when the
computation to generate the values are resource intensive and can be re-used when the key is the same.

This cache allows for recursive execution, meaning that the Compute method of the cache can be called from
inside the compute function given to the cache.  If doing so leads to a cycle (key A asks for key B asks for
key C asks for key A) an error will be returned.

In this example the cache is used to avoid long think times when calculating the value for the given key:

```go
func cacheExample() bool {
	thinkCache, _ := cache.NewComputeFunctionCache(func(key interface{}) (interface{}, error) {
		// lemme think for a second
		time.Sleep(1 * time.Second)

		// Sort of a silly computation!
		return key, nil

	})

	// First time it'll think for a second while computing the value
	nowTime := time.Now()
	val, _ := thinkCache.Compute(13)
	elapsedTime := time.Now().Sub(nowTime)
	
	fmt.Printf("Cache returned %v after %v\n", val, elapsedTime)

	// Second time it won't think, it'll take the value from the cache
	nowTime = time.Now()
	val, _ = thinkCache.Compute(13)
	elapsedTime = time.Now().Sub(nowTime)

	fmt.Printf("Cache returned %v the second time after %v\n", val, elapsedTime)

	return true
}
```

### CAR Cache

A CAR cache is a limited size in-memory computable cache.  CAR caches are like LRU caches
but have an algorithm that chooses values to be removed that is better performing (generally)
than LRU caches.  For more information see
[CAR Cache Algorithm](https://www.usenix.org/conference/fast-04/car-clock-adaptive-replacement)

When you create a CAR cache you provide a max number.  The number of values the CAR cache
will keep is max, but it might keep up to (2 * max) keys, as part of the algorithm keeps
a history of previous keys seen.

This cache allows for recursive execution, meaning that the Compute method of the cache can be called from
inside the compute function given to the cache.  If doing so leads to a cycle (key A asks for key B asks for
key C asks for key A) an error will be returned.

This example shows that one of the keys is dropped when the number of keys is greater than
max.  In this simple example the CAR cache behaves like an LRU.  The CAR cache starts performing
better when there are more a more complex set of cache hits.

[embedmd]:# (examples/car.go /^package.*/ $)
```go
package main

import (
	"fmt"
	"github.com/jwells131313/goethe/cache"
	"time"
)

func CARCache() {
	cCache, _ := cache.NewComputeFunctionCARCache(5, func(key interface{}) (interface{}, error) {
		// lemme think for a little bit
		time.Sleep(100 * time.Millisecond)

		// Sort of a silly computation!
		return key, nil

	})

	// One more than the cache can hold!
	for count := 0; count < 6; count++ {
		cCache.Compute(count)
	}

	for count := 0; count < 6; count++ {
		foundKey := cCache.HasKey(count)

		var result string
		if foundKey {
			result = "found"
		} else {
			result = "not found"
		}

		fmt.Printf("Key %d is %s\n", count, result)
	}

}
```

You can provide destructor functions to the CAR cache which will be invoked whenever the Compute method of
the cache removes a value.  This allows users to clean up any resources held in the values of the cache
when those values are released.  See [NewCARCacheWithDestructor](https://godoc.org/github.com/jwells131313/goethe/cache#NewCARCacheWithDestructor)
and [NewComputeFunctionCARCacheWithDestructor](https://godoc.org/github.com/jwells131313/goethe/cache#NewComputeFunctionCARCacheWithDestructor).

## Queues

[![GoDoc](https://godoc.org/github.com/jwells131313/goethe/queues?status.svg)](https://godoc.org/github.com/jwells131313/goethe/queues)

### Heap

A Heap is an ordered queue data structure where adding items to the queue is a O(log(n)) complexity operation
and removing items from the queue is a O(log(n)) complexity operation.  Peeking at the next item that would
be removed is O(1) and should be very quick.  The downside to heaps is that you cannot remove items from
the middle of the heap.  Also a lot of other priority queue algorithms have faster removal complexities.
However, it does guarantee a perfectly balanced search tree at all times and so in practice it can
often be faster than other ordered structures.

Heaps also support two items having keys with no ordering requirements between them.  So, for example
you can add integers 1, 3, 8, 3, 8, 5 and the result you would get from draining this heap would be
1, 3, 3, 5, 8, 8.  The 3's and 8's in this example could be returned in any order relative to each other.

This heap implementation is thread-safe (for any threads, not just Goethe threads) and keeps only a slice
of items of the cardinality of the number of items in the heap, so it's very good about memory space.

```go
func cmp(araw interface{}, braw interface{}) int {
	a := araw.(int)
	b := braw.(int)

	if a < b {
		return 1
	}
	if a == b {
		return 0
	}

	return -1
}

func BuildAHeap(t *testing.T) {
	heap := NewHeap(cmp)

	heap.Add(1)
	heap.Add(3)
	heap.Add(4)
	heap.Add(2)
	
	heap.Peek() // Would return 1
	
	heap.Get() // Would return 1
	heap.Get() // Would return 2
	heap.Get() // Would return 3
	heap.Get() // Would return 4
	
	heap.Peek() // Would return false, heap is empty
	heap.Get() // Would return false, heap is empyt
}
```

## Recursive Locks

In goethe threads you can have recursive read/write mutexes which obey the following rules:

* Only one writer lock is allowed into the critical section
* When holding a writer lock on a thread another writer lock may be acquired on the same thread (counting)
* When holding a writer lock on a thread you may acquire a reader lock on the same thread.  The writer lock remains in effect
* Many reader locks can be held on multiple different threads
* When holding a reader lock on a thread another reader lock may be acquired on the same thread (counting)
* When holding a reader lock you may not acquire a writer lock.  Doing so leads too easily to deadlocks
* When holding a writer lock you may acquire a reader lock
* Once a writer asks for the lock no more readers will be able to enter, so writers can starve readers
* There are TryReadLock and TryWriteLock methods which provide timeouts for acquiring the lock

The following is an example of a recursive write lock
```go
package main

import (
	"github.com/jwells131313/goethe"
)

var ethe = goethe.GG()
var lock = ethe.NewGoetheLock()

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

### Thread Local Storage

Goethe threads can take advantage of named thread local storage.  Thread local storage can be first
established by giving it a name, an initializer function and a destroyer function.  Then
in your goethe threads you can call GetThreadLocal with the name of your thread local and get
the type returned by the initializer.  Each thread has its own copy of the type so two threads
will not interfere with each other.  When the thread goes away the destructor for all named
thread local storage associated with that thread will be called.

The example in the Timers section below uses a thread local to get the
Timer object from inside the thread.

### Timer Heap

[![GoDoc](https://godoc.org/github.com/jwells131313/goethe/timers?status.svg)](https://godoc.org/github.com/jwells131313/goethe/timers)

A Timer Heap is a data structure of timers whose intent is to minimize the number of outstanding
timers being created by the system.  In general no matter how many jobs have been put onto a
Timer Heap there will be one outstanding running golang timer.

The function given to the timer heap to run will be run in goethe threads when their timer expires.

The following example prints out a friendly message after 1 millisecond!

```go
func ExampleTimerHeap() {
	// We don't use the error channel, but it's here for demonstration
	errChan := make(chan error)
	
	// Create a new timer!
	timer := timers.NewTimerHeap(errChan)
	
	// Don't forget to cancel the timer when you are done with the timer heap
	defer timer.Cancel()

	// Print Hello World after waiting one millisecond!
	timer.AddJobByDuration(time.Millisecond, func() error {
		fmt.Println("Hello World!")
		return nil
	})

	// Sleep 500 to give it plenty of time to print out
	time.Sleep(time.Duration(500) * time.Millisecond)

	// Output: Hello World!
}
```

### Timers

Goethe provides timer threads that run user code periodically.  There are two types of timers, one
with fixed delay and one with fixed rate.

A fixed delay timer will start the next timer once the user code has run to completion.  So
long running user code will cause the timer to not be invoked again until the user code has
completed and the delay period has passed again.  This timer is started with the
ThreadUtilities.ScheduleWithFixedDelay method

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

## Thread Pools

Thread pools use goethe threads so that you can use thread-ids, thread-locals and recursive
locks in your threaded application.

In order to create a thread pool you specify the min threads in the the pool, the max threads
in the pool and the decay time, which is the time a thread will be idle before being released
(if the number of threads is greater than the min).

You also give the pool an implementation of a FunctionQueue.  You can get a default
implementation of FunctionQueue from the goethe package.  However any implementation of
FunctionQueue will work, which allows you to use priority queues or other queue implementations.
Once a queue is associated with a pool you use the queue Enqueue API to give jobs to the thread pool.

You can also give the pool an ErrorQueue.  Any non-nil errors returned from the enqueued jobs will
be placed on the ErrorQueue.  It is up to the application to check and drain the ErrorQueue for
errors.

The following example uses recursive read/write locks, an error queue and a functional queue along
with a pool.  The work done in the randomWork method is just sleeping anywhere from 1 to 99
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

## Under Construction

In the future it is intended for goethe to provide the following:

* locks you can give up on after some duration
* Other queuing utilities

![](https://github.com/jwells131313/goethe/blob/master/images/goth.jpg "Go Thread Utilities")
